package core

import (
	"bytes"
	"context"
	_ "embed"
	"encoding/binary"
	"fmt"
	"math"
	"sort"
	"strings"
	"sync"
	"unicode"

	"github.com/chainreactors/malice-network/client/assets"
)

//go:embed search_index.bin
var embeddedSearchIndex []byte

// VectorIndex holds precomputed search vectors loaded from the embedded binary.
// Dual format: dense (neural embedding) + sparse (TF-IDF). Dense is used when
// the same embedding model is available at query time; sparse is the offline fallback.
type VectorIndex struct {
	mu        sync.RWMutex
	commands  []commandEntry
	byName    map[string]int
	hasDense  bool
	denseDim  int
	embModel  string         // model used for dense vectors
	vocab     map[string]int // TF-IDF
	idf       []float32
	catalog   string // compact "name: desc" for LLM query rewriting
}

type commandEntry struct {
	Name, Type, Category, Source, Description string
	Dense                                     []float32
	Sparse                                    []sparseEntry
}

type sparseEntry struct {
	Index uint16
	Value float32
}

type scored struct {
	idx   int
	score float64
}

// SemanticResult is a search hit from vector similarity.
type SemanticResult struct {
	Name, Type, Category, Source, Description string
	Score                                     float64
}

// NewVectorIndex deserializes the embedded binary. Pure data load, no computation.
func NewVectorIndex() *VectorIndex {
	vi := &VectorIndex{byName: make(map[string]int), vocab: make(map[string]int)}
	if len(embeddedSearchIndex) < 16 {
		return vi
	}

	r := bytes.NewReader(embeddedSearchIndex)
	var magic [4]byte
	binary.Read(r, binary.LittleEndian, &magic)
	if string(magic[:]) != "SRCH" {
		return vi
	}

	var version, numCmds, vocabSize, denseDim uint32
	binary.Read(r, binary.LittleEndian, &version)
	binary.Read(r, binary.LittleEndian, &numCmds)
	binary.Read(r, binary.LittleEndian, &vocabSize)
	binary.Read(r, binary.LittleEndian, &denseDim)
	if version >= 3 {
		vi.embModel = readStr(r)
	}
	vi.hasDense = denseDim > 0
	vi.denseDim = int(denseDim)

	// TF-IDF vocab
	vi.idf = make([]float32, vocabSize)
	for i := 0; i < int(vocabSize); i++ {
		term := readStr(r)
		binary.Read(r, binary.LittleEndian, &vi.idf[i])
		vi.vocab[term] = i
	}

	// Commands
	vi.commands = make([]commandEntry, numCmds)
	var cat strings.Builder
	for i := range vi.commands {
		cmd := &vi.commands[i]
		cmd.Name = readStr(r)
		var tb uint8
		binary.Read(r, binary.LittleEndian, &tb)
		cmd.Type = "command"
		if tb == 1 {
			cmd.Type = "plugin"
		}
		cmd.Category = readStr(r)
		cmd.Source = readStr(r)
		cmd.Description = readStr(r)

		var nz uint16
		binary.Read(r, binary.LittleEndian, &nz)
		cmd.Sparse = make([]sparseEntry, nz)
		for j := range cmd.Sparse {
			binary.Read(r, binary.LittleEndian, &cmd.Sparse[j].Index)
			binary.Read(r, binary.LittleEndian, &cmd.Sparse[j].Value)
		}
		if vi.hasDense {
			cmd.Dense = make([]float32, denseDim)
			for j := range cmd.Dense {
				binary.Read(r, binary.LittleEndian, &cmd.Dense[j])
			}
		}

		vi.byName[cmd.Name] = i
		d := cmd.Description
		if len(d) > 60 {
			d = d[:60]
		}
		fmt.Fprintf(&cat, "%s: %s\n", cmd.Name, d)
	}
	vi.catalog = cat.String()
	return vi
}

func readStr(r *bytes.Reader) string {
	var n uint16
	binary.Read(r, binary.LittleEndian, &n)
	b := make([]byte, n)
	r.Read(b)
	return string(b)
}

func (vi *VectorIndex) IsDense() bool        { return vi.hasDense }
func (vi *VectorIndex) Len() int             { vi.mu.RLock(); defer vi.mu.RUnlock(); return len(vi.commands) }
func (vi *VectorIndex) EmbeddingModel() string { return vi.embModel }

// Search tries dense (neural) first, then falls back to sparse (TF-IDF).
func (vi *VectorIndex) Search(ctx context.Context, query, typeFilter, category string, limit int) ([]SemanticResult, error) {
	vi.mu.RLock()
	defer vi.mu.RUnlock()
	if len(vi.commands) == 0 {
		return nil, nil
	}

	// Dense: if embedding API available and model matches
	if vi.hasDense {
		if results := vi.denseSearch(ctx, query, typeFilter, category, limit); len(results) > 0 {
			return results, nil
		}
	}
	// Sparse fallback (always works offline)
	return vi.sparseSearch(ctx, query, typeFilter, category, limit), nil
}

func (vi *VectorIndex) denseSearch(ctx context.Context, query, typeFilter, category string, limit int) []SemanticResult {
	ai, err := assets.GetValidAISettings()
	if err != nil {
		return nil
	}
	if vi.embModel != "" && ai.EmbeddingModel != "" && ai.EmbeddingModel != vi.embModel {
		return nil
	}
	qv, err := NewAIClient(ai).GetEmbedding(ctx, query)
	if err != nil {
		return nil
	}
	return vi.searchVec(qv, typeFilter, category, limit, 0.3)
}

func (vi *VectorIndex) sparseSearch(ctx context.Context, query, typeFilter, category string, limit int) []SemanticResult {
	terms := query
	if ai, err := assets.GetValidAISettings(); err == nil {
		if rewritten, err := rewriteQuery(ctx, ai, query, vi.catalog); err == nil && rewritten != "" {
			terms = query + " " + rewritten
		}
	}
	qv := vi.tfidfTransform(terms)
	threshold := 0.15
	if containsCJK(query) {
		threshold = 0.01
	}
	var hits []scored
	for i, cmd := range vi.commands {
		if typeFilter != "" && cmd.Type != typeFilter {
			continue
		}
		if category != "" && cmd.Category != category {
			continue
		}
		if s := sparseDot(qv, cmd.Sparse); s > threshold {
			hits = append(hits, scored{i, s})
		}
	}
	return vi.toResults(hits, limit)
}

// SearchWithVec searches using a pre-computed dense query vector.
func (vi *VectorIndex) SearchWithVec(qv []float32, typeFilter, category string, limit int) []SemanticResult {
	vi.mu.RLock()
	defer vi.mu.RUnlock()
	return vi.searchVec(qv, typeFilter, category, limit, 0.3)
}

func (vi *VectorIndex) searchVec(qv []float32, typeFilter, category string, limit int, threshold float64) []SemanticResult {
	var hits []scored
	for i, cmd := range vi.commands {
		if typeFilter != "" && cmd.Type != typeFilter {
			continue
		}
		if category != "" && cmd.Category != category {
			continue
		}
		if s := cosine(qv, cmd.Dense); s > threshold {
			hits = append(hits, scored{i, s})
		}
	}
	return vi.toResults(hits, limit)
}

func (vi *VectorIndex) toResults(hits []scored, limit int) []SemanticResult {
	sort.Slice(hits, func(a, b int) bool { return hits[a].score > hits[b].score })
	if limit <= 0 {
		limit = 20
	}
	if len(hits) > limit {
		hits = hits[:limit]
	}
	out := make([]SemanticResult, len(hits))
	for i, h := range hits {
		c := vi.commands[h.idx]
		out[i] = SemanticResult{c.Name, c.Type, c.Category, c.Source, c.Description, h.score}
	}
	return out
}

func (vi *VectorIndex) tfidfTransform(text string) []sparseEntry {
	terms := Tokenize(text)
	tf := make(map[string]int)
	for _, t := range terms {
		tf[t]++
	}
	total := float64(len(terms))
	if total == 0 {
		return nil
	}
	var entries []sparseEntry
	var normSq float64
	for term, count := range tf {
		if idx, ok := vi.vocab[term]; ok {
			v := float32(float64(count)/total) * vi.idf[idx]
			entries = append(entries, sparseEntry{uint16(idx), v})
			normSq += float64(v) * float64(v)
		}
	}
	if normSq > 0 {
		norm := float32(math.Sqrt(normSq))
		for i := range entries {
			entries[i].Value /= norm
		}
	}
	sort.Slice(entries, func(a, b int) bool { return entries[a].Index < entries[b].Index })
	return entries
}

// HybridSearch merges vector semantic results with FTS5 keyword results.
func HybridSearch(ctx context.Context, si *SearchIndex, vi *VectorIndex, query, typeFilter, category string, limit int) ([]SearchResult, error) {
	if limit <= 0 {
		limit = 20
	}
	type entry struct {
		result SearchResult
		score  float64
	}
	seen := make(map[string]*entry)

	if vi != nil && vi.Len() > 0 {
		if results, _ := vi.Search(ctx, query, typeFilter, category, limit); len(results) > 0 {
			for _, r := range results {
				seen[r.Name] = &entry{
					result: SearchResult{Name: r.Name, Type: r.Type, Category: r.Category,
						Source: r.Source, Description: r.Description, Snippet: r.Description},
					score: r.Score,
				}
			}
		}
	}

	if si != nil {
		if results, _ := si.Search(query, typeFilter, category, limit); len(results) > 0 {
			for _, r := range results {
				if e, ok := seen[r.Name]; ok {
					e.score += 0.1
				} else {
					s := 0.0
					if r.Rank < 0 {
						s = 0.3 / (1.0 + math.Abs(r.Rank))
					}
					seen[r.Name] = &entry{result: r, score: s}
				}
			}
		}
	}

	if len(seen) == 0 {
		return nil, nil
	}
	finals := make([]entry, 0, len(seen))
	for _, e := range seen {
		finals = append(finals, *e)
	}
	sort.Slice(finals, func(a, b int) bool { return finals[a].score > finals[b].score })
	if len(finals) > limit {
		finals = finals[:limit]
	}
	out := make([]SearchResult, len(finals))
	for i := range finals {
		out[i] = finals[i].result
	}
	return out, nil
}

func rewriteQuery(ctx context.Context, ai *assets.AISettings, query, catalog string) (string, error) {
	prompt := fmt.Sprintf(`Rewrite this query into C2 command search keywords (one line, no explanation, include Chinese+English terms):
Commands: %s
Query: %s
Keywords:`, catalog, query)
	result, err := NewAIClient(ai).askOpenAIWith(ctx, "", prompt, 200, 0.1)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(result), nil
}

// --- Math helpers ---

func cosine(a, b []float32) float64 {
	if len(a) != len(b) || len(a) == 0 {
		return 0
	}
	var dot, na, nb float64
	for i := range a {
		fa, fb := float64(a[i]), float64(b[i])
		dot += fa * fb
		na += fa * fa
		nb += fb * fb
	}
	if d := math.Sqrt(na) * math.Sqrt(nb); d > 0 {
		return dot / d
	}
	return 0
}

func sparseDot(a, b []sparseEntry) float64 {
	ai, bi := 0, 0
	var dot float64
	for ai < len(a) && bi < len(b) {
		if a[ai].Index == b[bi].Index {
			dot += float64(a[ai].Value) * float64(b[bi].Value)
			ai++
			bi++
		} else if a[ai].Index < b[bi].Index {
			ai++
		} else {
			bi++
		}
	}
	return dot
}

// --- Tokenizer (used by TF-IDF at runtime and genembeddings at build time) ---

// Tokenize splits text into lowercase terms with CJK unigram+bigram handling.
func Tokenize(text string) []string {
	text = strings.ToLower(text)
	var tokens []string
	var word strings.Builder

	flush := func() {
		if word.Len() > 0 {
			tokens = append(tokens, word.String())
			word.Reset()
		}
	}

	for _, r := range text {
		if isCJK(r) {
			flush()
			tokens = append(tokens, string(r))
		} else if unicode.IsLetter(r) || unicode.IsDigit(r) {
			word.WriteRune(r)
		} else {
			flush()
		}
	}
	flush()

	// CJK bigrams
	n := len(tokens)
	for i := 0; i < n-1; i++ {
		if containsCJK(tokens[i]) && containsCJK(tokens[i+1]) {
			tokens = append(tokens, tokens[i]+tokens[i+1])
		}
	}
	return tokens
}

func isCJK(r rune) bool {
	return unicode.Is(unicode.Han, r) || unicode.Is(unicode.Hangul, r) ||
		unicode.Is(unicode.Katakana, r) || unicode.Is(unicode.Hiragana, r)
}

func containsCJK(s string) bool {
	for _, r := range s {
		if isCJK(r) {
			return true
		}
	}
	return false
}

// BuildCommandText is used by genembeddings to create indexable text from command metadata.
func BuildCommandText(name, short, long, usage, example string, flags []string) string {
	parts := []string{name}
	if short != "" {
		parts = append(parts, short)
	}
	if long != "" {
		parts = append(parts, long)
	}
	if usage != "" {
		parts = append(parts, usage)
	}
	if example != "" {
		parts = append(parts, example)
	}
	if len(flags) > 0 {
		parts = append(parts, strings.Join(flags, " "))
	}
	return strings.Join(parts, "\n")
}
