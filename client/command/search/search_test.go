package search

import (
	"encoding/json"
	"path/filepath"
	"testing"

	"github.com/carapace-sh/carapace"
	"github.com/chainreactors/malice-network/client/core"
	"github.com/spf13/cobra"
)

func TestSearchGroupCompleterUsesSearchIndexCategories(t *testing.T) {
	dir := t.TempDir()
	si, err := core.NewSearchIndex(filepath.Join(dir, "search.db"))
	if err != nil {
		t.Fatalf("NewSearchIndex: %v", err)
	}
	t.Cleanup(func() { si.Close() })

	if err := si.Rebuild(func() []*cobra.Command {
		return []*cobra.Command{
			{Use: "ls", Short: "List files", GroupID: "file", Annotations: map[string]string{"source": "builtin"}},
			{Use: "ps", Short: "List processes", GroupID: "sys", Annotations: map[string]string{"source": "builtin"}},
		}
	}); err != nil {
		t.Fatalf("Rebuild: %v", err)
	}

	values := completionValues(t, searchGroupCompleter(&core.Console{SearchIndex: si}))

	if !hasCompletionValue(values, "file") {
		t.Fatalf("completion values = %#v, want file", values)
	}
	if !hasCompletionValue(values, "sys") {
		t.Fatalf("completion values = %#v, want sys", values)
	}
}

func completionValues(t testing.TB, action carapace.Action) []string {
	t.Helper()

	data, err := json.Marshal(action.Invoke(carapace.Context{}))
	if err != nil {
		t.Fatalf("marshal completion action failed: %v", err)
	}
	var decoded struct {
		Values []struct {
			Value string `json:"value"`
		} `json:"values"`
	}
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("decode completion action failed: %v", err)
	}
	values := make([]string, 0, len(decoded.Values))
	for _, value := range decoded.Values {
		values = append(values, value.Value)
	}
	return values
}

func hasCompletionValue(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}
