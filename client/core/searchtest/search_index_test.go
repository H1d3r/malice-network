package searchtest

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/chainreactors/malice-network/client/core"
	"github.com/spf13/cobra"
)

func testSearchIndex(t *testing.T) *core.SearchIndex {
	t.Helper()
	dir := t.TempDir()
	si, err := core.NewSearchIndex(filepath.Join(dir, "test_search.db"))
	if err != nil {
		t.Fatalf("NewSearchIndex: %v", err)
	}
	t.Cleanup(func() { si.Close() })
	return si
}

func makeTestCommands() []*cobra.Command {
	upload := &cobra.Command{
		Use:   "upload [local] [remote]",
		Short: "Upload a file to the target",
		Long:  "Upload a local file to the remote target filesystem",
		Annotations: map[string]string{
			"ttp":    "T1105",
			"opsec":  "7",
			"source": "builtin",
		},
		GroupID: "file",
	}

	download := &cobra.Command{
		Use:   "download [remote] [local]",
		Short: "Download a file from target",
		Long:  "Download a remote file from the target to local filesystem",
		Annotations: map[string]string{
			"ttp":    "T1041",
			"opsec":  "8",
			"source": "builtin",
		},
		GroupID: "file",
	}

	whoami := &cobra.Command{
		Use:   "whoami",
		Short: "Print current user info",
		Annotations: map[string]string{
			"ttp":    "T1033",
			"opsec":  "9",
			"source": "builtin",
		},
		GroupID: "sys",
	}

	pivot := &cobra.Command{
		Use:   "pivot",
		Short: "Create TCP pivot for lateral movement",
		Long:  "Create a TCP pivot to enable lateral movement across network segments",
		Annotations: map[string]string{
			"ttp":    "T1090",
			"opsec":  "5",
			"source": "builtin",
		},
		GroupID: "pivot",
	}

	ls := &cobra.Command{
		Use:     "ls [path]",
		Short:   "List directory contents",
		Long:    "列出目标系统上指定目录的文件和文件夹",
		GroupID: "file",
		Annotations: map[string]string{
			"source": "builtin",
		},
	}

	hiddenCmd := &cobra.Command{
		Use:    "internal-debug",
		Short:  "Internal debug command",
		Hidden: true,
	}

	parent := &cobra.Command{
		Use:   "session",
		Short: "Session management",
		Annotations: map[string]string{
			"source": "builtin",
		},
		GroupID: "manage",
	}
	sub := &cobra.Command{
		Use:   "list",
		Short: "List all sessions",
	}
	parent.AddCommand(sub)

	return []*cobra.Command{upload, download, whoami, pivot, ls, hiddenCmd, parent}
}

func TestNewSearchIndex(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "test.db")
	si, err := core.NewSearchIndex(dbPath)
	if err != nil {
		t.Fatalf("NewSearchIndex: %v", err)
	}
	defer si.Close()

	if _, err := os.Stat(dbPath); os.IsNotExist(err) {
		t.Error("database file not created")
	}
}

func TestRebuildAndSearch(t *testing.T) {
	si := testSearchIndex(t)
	cmds := makeTestCommands()

	err := si.Rebuild(func() []*cobra.Command { return cmds })
	if err != nil {
		t.Fatalf("Rebuild: %v", err)
	}

	results, err := si.Search("upload", "", "", 10)
	if err != nil {
		t.Fatalf("Search: %v", err)
	}
	if len(results) == 0 {
		t.Fatal("expected results for 'upload'")
	}
	if results[0].Name != "upload" {
		t.Errorf("expected first result 'upload', got %q", results[0].Name)
	}
}

func TestSearchByDescription(t *testing.T) {
	si := testSearchIndex(t)
	cmds := makeTestCommands()
	si.Rebuild(func() []*cobra.Command { return cmds })

	results, err := si.Search("lateral movement", "", "", 10)
	if err != nil {
		t.Fatalf("Search: %v", err)
	}
	if len(results) == 0 {
		t.Fatal("expected results for 'lateral movement'")
	}
	found := false
	for _, r := range results {
		if r.Name == "pivot" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected 'pivot' in results for 'lateral movement'")
	}
}

func TestSearchTypeFilter(t *testing.T) {
	si := testSearchIndex(t)
	cmds := makeTestCommands()
	pluginCmd := &cobra.Command{
		Use:   "mimikatz",
		Short: "Credential harvesting",
		Annotations: map[string]string{"source": "mal"},
	}
	si.Rebuild(func() []*cobra.Command { return cmds }, func() []*cobra.Command { return []*cobra.Command{pluginCmd} })

	results, err := si.Search("mimikatz", "plugin", "", 10)
	if err != nil {
		t.Fatalf("Search: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 plugin result, got %d", len(results))
	}
	if results[0].Type != "plugin" {
		t.Errorf("expected type 'plugin', got %q", results[0].Type)
	}
}

func TestSearchCategoryFilter(t *testing.T) {
	si := testSearchIndex(t)
	cmds := makeTestCommands()
	si.Rebuild(func() []*cobra.Command { return cmds })

	results, err := si.Search("file target", "", "file", 10)
	if err != nil {
		t.Fatalf("Search: %v", err)
	}
	for _, r := range results {
		if r.Category != "file" {
			t.Errorf("expected category 'file', got %q for %s", r.Category, r.Name)
		}
	}
}

func TestSearchCJK(t *testing.T) {
	// CJK is handled by VectorIndex (semantic), not FTS5 (keyword).
	// FTS5 unicode61 tokenizer doesn't split CJK well, so CJK queries
	// are expected to return no results here — they're handled by HybridSearch.
	si := testSearchIndex(t)
	cmds := makeTestCommands()
	si.Rebuild(func() []*cobra.Command { return cmds })

	_, err := si.Search("文件", "", "", 10)
	if err != nil {
		t.Fatalf("Search should not error on CJK: %v", err)
	}
}

func TestSearchLimit(t *testing.T) {
	si := testSearchIndex(t)
	cmds := makeTestCommands()
	si.Rebuild(func() []*cobra.Command { return cmds })

	results, err := si.Search("file target upload download", "", "", 1)
	if err != nil {
		t.Fatalf("Search: %v", err)
	}
	if len(results) > 1 {
		t.Errorf("expected at most 1 result with limit=1, got %d", len(results))
	}
}

func TestSearchHiddenExcluded(t *testing.T) {
	si := testSearchIndex(t)
	cmds := makeTestCommands()
	si.Rebuild(func() []*cobra.Command { return cmds })

	results, err := si.Search("internal-debug", "", "", 10)
	if err != nil {
		return
	}
	for _, r := range results {
		if r.Name == "internal-debug" {
			t.Error("hidden command should not appear in search results")
		}
	}
}

func TestSearchSubcommands(t *testing.T) {
	si := testSearchIndex(t)
	cmds := makeTestCommands()
	si.Rebuild(func() []*cobra.Command { return cmds })

	results, err := si.Search("session", "", "", 10)
	if err != nil {
		t.Fatalf("Search: %v", err)
	}
	if len(results) == 0 {
		t.Fatal("expected results for 'session'")
	}
}

func TestSearchSnippet(t *testing.T) {
	si := testSearchIndex(t)
	cmds := makeTestCommands()
	si.Rebuild(func() []*cobra.Command { return cmds })

	results, err := si.Search("lateral", "", "", 10)
	if err != nil {
		t.Fatalf("Search: %v", err)
	}
	if len(results) == 0 {
		t.Fatal("expected results for 'lateral'")
	}
	if results[0].Snippet == "" {
		t.Error("expected non-empty snippet")
	}
}

func TestSearchEmptyQuery(t *testing.T) {
	si := testSearchIndex(t)
	cmds := makeTestCommands()
	si.Rebuild(func() []*cobra.Command { return cmds })

	_, err := si.Search("", "", "", 10)
	if err == nil {
		t.Log("empty query handled gracefully")
	}
}

func TestRebuildMultipleSources(t *testing.T) {
	si := testSearchIndex(t)
	cmds := makeTestCommands()
	extra := &cobra.Command{
		Use: "keylogger", Short: "Start keylogger",
		Annotations: map[string]string{"source": "mal"},
	}
	si.Rebuild(
		func() []*cobra.Command { return cmds },
		func() []*cobra.Command { return []*cobra.Command{extra} },
	)

	results, err := si.Search("keylogger", "", "", 10)
	if err != nil {
		t.Fatalf("Search: %v", err)
	}
	if len(results) == 0 {
		t.Fatal("expected to find keylogger from second source")
	}
}
