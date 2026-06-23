// genembeddings generates search_index.bin — precomputed TF-IDF + optional neural embedding vectors.
//
//	go run ./client/cmd/genembeddings/
//	go run ./client/cmd/genembeddings/ --embedding-url https://api.kimi.com/coding --embedding-key <key>
package main

import (
	"bytes"
	"context"
	"encoding/binary"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"math"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/chainreactors/malice-network/client/core"
)

type indexedCmd struct {
	name, category, source, desc string
	plugin                       bool
	dense                        []float32
	sparse                       []sparseVec
}

type sparseVec struct {
	index uint16
	value float32
}

func main() {
	embURL := flag.String("embedding-url", "", "Embedding API base URL")
	embKey := flag.String("embedding-key", "", "Embedding API key")
	embModel := flag.String("embedding-model", "moonshot-v1-embedding", "Embedding model")
	docsDir := flag.String("docs", "docs/reference/commands", "Command reference docs")
	luaDir := flag.String("lua", "helper/intl/community/community/modules", "Embedded Lua modules")
	malCommunity := flag.String("mal-community", "", "Cloned mal-community repo")
	output := flag.String("output", "client/core/search_index.bin", "Output path")
	batchSize := flag.Int("batch", 16, "Embedding batch size")
	flag.Parse()

	// --- Collect commands ---
	var sections []cmdSection
	for _, f := range []struct{ name, src string }{
		{"client.md", "builtin"}, {"implant.md", "builtin"}, {"community.md", "mal"},
	} {
		if data, err := os.ReadFile(filepath.Join(*docsDir, f.name)); err == nil {
			sections = append(sections, parseMD(string(data), f.src)...)
		}
	}
	if *luaDir != "" {
		if cmds, err := parseLuaDir(*luaDir); err == nil {
			fmt.Fprintf(os.Stderr, "embedded Lua: %d commands\n", len(cmds))
			for _, c := range cmds {
				sections = append(sections, cmdSection{c.name, "community", "mal", c.description, luaText(c)})
			}
		}
	}
	if *malCommunity != "" {
		if cmds, err := parseMalCommunityRepo(*malCommunity); err == nil {
			fmt.Fprintf(os.Stderr, "mal-community: %d commands\n", len(cmds))
			for _, c := range cmds {
				src := c.source
				if src == "" {
					src = "mal-community"
				}
				sections = append(sections, cmdSection{c.name, "community", src, c.description, luaText(c)})
			}
		}
	}

	// Dedup (last wins)
	seen := make(map[string]bool)
	var unique []cmdSection
	for i := len(sections) - 1; i >= 0; i-- {
		if !seen[sections[i].name] {
			seen[sections[i].name] = true
			unique = append(unique, sections[i])
		}
	}
	sections = unique
	fmt.Fprintf(os.Stderr, "total: %d commands\n", len(sections))

	texts := make([]string, len(sections))
	commands := make([]indexedCmd, len(sections))
	for i, s := range sections {
		texts[i] = s.text
		commands[i] = indexedCmd{name: s.name, category: s.category, source: s.source,
			desc: s.desc, plugin: s.source != "builtin"}
	}

	// --- TF-IDF (always) ---
	vocab, idf := buildTFIDF(texts)
	fmt.Fprintf(os.Stderr, "TF-IDF: %d terms\n", len(vocab))
	for i := range commands {
		commands[i].sparse = tfidfVec(texts[i], vocab, idf)
	}

	// --- Neural embedding (optional) ---
	modelName := ""
	if *embURL != "" && *embKey != "" {
		modelName = *embModel
		fmt.Fprintf(os.Stderr, "embedding: %s (%s)\n", *embURL, *embModel)
		endpoint := strings.TrimSuffix(*embURL, "/") + "/v1/embeddings"
		for i := 0; i < len(texts); i += *batchSize {
			end := min(i+*batchSize, len(texts))
			vecs, err := embed(context.Background(), endpoint, *embKey, *embModel, texts[i:end])
			if err != nil {
				fmt.Fprintf(os.Stderr, "  batch %d-%d failed: %v\n", i, end, err)
				os.Exit(1)
			}
			for j, v := range vecs {
				commands[i+j].dense = v
			}
			fmt.Fprintf(os.Stderr, "  %d/%d\n", end, len(texts))
			if end < len(texts) {
				time.Sleep(200 * time.Millisecond)
			}
		}
	}

	// --- Write binary ---
	denseDim := 0
	if len(commands) > 0 && len(commands[0].dense) > 0 {
		denseDim = len(commands[0].dense)
	}

	var buf bytes.Buffer
	buf.Write([]byte("SRCH"))
	binary.Write(&buf, binary.LittleEndian, uint32(3))
	binary.Write(&buf, binary.LittleEndian, uint32(len(commands)))
	binary.Write(&buf, binary.LittleEndian, uint32(len(vocab)))
	binary.Write(&buf, binary.LittleEndian, uint32(denseDim))
	writeStr(&buf, modelName)

	sortedVocab := make([]string, len(vocab))
	for t, i := range vocab {
		sortedVocab[i] = t
	}
	for i, t := range sortedVocab {
		writeStr(&buf, t)
		binary.Write(&buf, binary.LittleEndian, idf[i])
	}

	for _, cmd := range commands {
		writeStr(&buf, cmd.name)
		t := uint8(0)
		if cmd.plugin {
			t = 1
		}
		binary.Write(&buf, binary.LittleEndian, t)
		writeStr(&buf, cmd.category)
		writeStr(&buf, cmd.source)
		writeStr(&buf, cmd.desc)
		binary.Write(&buf, binary.LittleEndian, uint16(len(cmd.sparse)))
		for _, sv := range cmd.sparse {
			binary.Write(&buf, binary.LittleEndian, sv.index)
			binary.Write(&buf, binary.LittleEndian, sv.value)
		}
		for _, v := range cmd.dense {
			binary.Write(&buf, binary.LittleEndian, v)
		}
	}

	os.WriteFile(*output, buf.Bytes(), 0644)
	mode := "sparse-only"
	if denseDim > 0 {
		mode = fmt.Sprintf("dual (sparse + %d-dim dense)", denseDim)
	}
	fmt.Fprintf(os.Stderr, "wrote %d commands, %s (%d bytes) → %s\n", len(commands), mode, buf.Len(), *output)
}

// --- Helpers ---

func writeStr(buf *bytes.Buffer, s string) {
	b := []byte(s)
	binary.Write(buf, binary.LittleEndian, uint16(len(b)))
	buf.Write(b)
}

func buildTFIDF(docs []string) (map[string]int, []float32) {
	vocab := make(map[string]int)
	df := make(map[string]int)
	for _, doc := range docs {
		seen := make(map[string]bool)
		for _, t := range core.Tokenize(doc) {
			if _, ok := vocab[t]; !ok {
				vocab[t] = len(vocab)
			}
			if !seen[t] {
				df[t]++
				seen[t] = true
			}
		}
	}
	n := float64(len(docs))
	idf := make([]float32, len(vocab))
	for t, i := range vocab {
		idf[i] = float32(math.Log(1 + n/(1+float64(df[t]))))
	}
	return vocab, idf
}

func tfidfVec(text string, vocab map[string]int, idf []float32) []sparseVec {
	terms := core.Tokenize(text)
	tf := make(map[string]int)
	for _, t := range terms {
		tf[t]++
	}
	total := float64(len(terms))
	if total == 0 {
		return nil
	}
	var out []sparseVec
	var normSq float64
	for t, c := range tf {
		if i, ok := vocab[t]; ok {
			v := float32(float64(c)/total) * idf[i]
			out = append(out, sparseVec{uint16(i), v})
			normSq += float64(v) * float64(v)
		}
	}
	if normSq > 0 {
		norm := float32(math.Sqrt(normSq))
		for i := range out {
			out[i].value /= norm
		}
	}
	sort.Slice(out, func(a, b int) bool { return out[a].index < out[b].index })
	return out
}

func embed(ctx context.Context, endpoint, key, model string, texts []string) ([][]float32, error) {
	body, _ := json.Marshal(map[string]interface{}{"model": model, "input": texts})
	req, _ := http.NewRequestWithContext(ctx, "POST", endpoint, bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+key)
	resp, err := (&http.Client{Timeout: 120 * time.Second}).Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	rb, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 300 {
		return nil, fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(rb[:min(200, len(rb))]))
	}
	var r struct {
		Data []struct {
			Embedding []float32 `json:"embedding"`
			Index     int       `json:"index"`
		} `json:"data"`
	}
	json.Unmarshal(rb, &r)
	out := make([][]float32, len(texts))
	for _, d := range r.Data {
		if d.Index < len(out) {
			out[d.Index] = d.Embedding
		}
	}
	return out, nil
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
