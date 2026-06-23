package searchtest

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/chainreactors/malice-network/client/core"
	"github.com/spf13/cobra"
)

func TestVectorIndexLoadsEmbedded(t *testing.T) {
	vi := core.NewVectorIndex()
	if vi.Len() == 0 {
		t.Skip("no precomputed index — run genembeddings first")
	}
	t.Logf("loaded %d commands from binary index", vi.Len())
}

func TestVectorIndexSearch(t *testing.T) {
	vi := core.NewVectorIndex()
	if vi.Len() == 0 {
		t.Skip("no precomputed index")
	}

	results, err := vi.Search(context.Background(), "upload file", "", "", 10)
	if err != nil {
		t.Fatalf("search error: %v", err)
	}
	if len(results) == 0 {
		// Dense mode requires embedding API at query time — skip if not configured
		t.Skip("no results (dense mode requires embedding API for query vectorization)")
	}
	t.Logf("'upload file' → %d results, top: %s (%.3f)", len(results), results[0].Name, results[0].Score)
}

func TestVectorIndexSearchCJK(t *testing.T) {
	vi := core.NewVectorIndex()
	if vi.Len() == 0 {
		t.Skip("no precomputed index")
	}

	results, err := vi.Search(context.Background(), "提权", "", "", 10)
	if err != nil {
		t.Fatalf("search error: %v", err)
	}
	if len(results) > 0 {
		t.Logf("'提权' → %d results, top: %s (%.3f)", len(results), results[0].Name, results[0].Score)
	}
}

func TestHybridSearchFTSOnly(t *testing.T) {
	dir := t.TempDir()
	si, err := core.NewSearchIndex(filepath.Join(dir, "test.db"))
	if err != nil {
		t.Fatalf("NewSearchIndex: %v", err)
	}
	defer si.Close()

	cmds := makeTestCommands()
	si.Rebuild(func() []*cobra.Command { return cmds })

	vi := core.NewVectorIndex()
	results, err := core.HybridSearch(context.Background(), si, vi, "upload", "", "", 10)
	if err != nil {
		t.Fatalf("HybridSearch: %v", err)
	}
	if len(results) == 0 {
		t.Fatal("expected results from hybrid search")
	}
}

func TestHybridSearchNilVector(t *testing.T) {
	dir := t.TempDir()
	si, err := core.NewSearchIndex(filepath.Join(dir, "test.db"))
	if err != nil {
		t.Fatalf("NewSearchIndex: %v", err)
	}
	defer si.Close()

	cmds := makeTestCommands()
	si.Rebuild(func() []*cobra.Command { return cmds })

	results, err := core.HybridSearch(context.Background(), si, nil, "upload", "", "", 10)
	if err != nil {
		t.Fatalf("HybridSearch: %v", err)
	}
	if len(results) == 0 {
		t.Fatal("expected FTS5 results with nil vector index")
	}
}

func TestHybridSearchSemantic(t *testing.T) {
	dir := t.TempDir()
	si, err := core.NewSearchIndex(filepath.Join(dir, "test.db"))
	if err != nil {
		t.Fatalf("NewSearchIndex: %v", err)
	}
	defer si.Close()

	cmds := makeTestCommands()
	si.Rebuild(func() []*cobra.Command { return cmds })

	vi := core.NewVectorIndex()
	if vi.Len() == 0 {
		t.Skip("no precomputed index")
	}

	results, err := core.HybridSearch(context.Background(), si, vi, "privilege escalation", "", "", 10)
	if err != nil {
		t.Fatalf("HybridSearch: %v", err)
	}
	t.Logf("'privilege escalation' → %d results", len(results))
	for i, r := range results {
		if i >= 3 {
			break
		}
		t.Logf("  %d. %s — %s", i+1, r.Name, r.Description)
	}
}

func TestBuildCommandText(t *testing.T) {
	text := core.BuildCommandText("upload", "Upload a file", "Long description", "upload [file]", "", []string{"force", "timeout"})
	if text == "" {
		t.Fatal("BuildCommandText returned empty")
	}
}
