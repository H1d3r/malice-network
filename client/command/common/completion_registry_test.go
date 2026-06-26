package common

import (
	"testing"

	"github.com/carapace-sh/carapace"
	"github.com/spf13/cobra"
)

func TestFlagCompletionRegistryRoundTrip(t *testing.T) {
	cmd := &cobra.Command{Use: "test"}
	cmd.Flags().String("listener", "", "listener id")
	cmd.Flags().String("name", "", "name")

	comps := carapace.ActionMap{
		"listener": carapace.ActionValues("lns-a", "lns-b"),
	}
	RegisterFlagCompletions(cmd, comps)

	action, ok := GetFlagCompletion(cmd, "listener")
	if !ok {
		t.Fatal("expected listener completion to be registered")
	}
	invoked := action.Invoke(carapace.Context{})
	data, err := invoked.MarshalJSON()
	if err != nil {
		t.Fatalf("MarshalJSON: %v", err)
	}
	if len(data) == 0 {
		t.Fatal("expected non-empty completion data")
	}

	_, ok = GetFlagCompletion(cmd, "name")
	if ok {
		t.Fatal("name should not have a registered completion")
	}
}

func TestArgCompletionRegistryRoundTrip(t *testing.T) {
	cmd := &cobra.Command{Use: "test"}

	actions := []carapace.Action{
		carapace.ActionValues("session-1", "session-2"),
	}
	RegisterArgCompletions(cmd, nil, actions)

	action, ok := GetArgCompletion(cmd, 0)
	if !ok {
		t.Fatal("expected arg[0] completion to be registered")
	}
	invoked := action.Invoke(carapace.Context{})
	data, _ := invoked.MarshalJSON()
	if len(data) == 0 {
		t.Fatal("expected non-empty completion data")
	}

	_, ok = GetArgCompletion(cmd, 1)
	if ok {
		t.Fatal("arg[1] should not be registered")
	}
}

func TestArgCompletionRegistryFallsBackToAnyAction(t *testing.T) {
	cmd := &cobra.Command{Use: "test"}
	anyAction := carapace.ActionValues("path-a", "path-b")

	RegisterArgCompletions(cmd, &anyAction, nil)

	action, ok := GetArgCompletion(cmd, 3)
	if !ok {
		t.Fatal("expected positional-any completion to be registered")
	}
	invoked := action.Invoke(carapace.Context{})
	data, err := invoked.MarshalJSON()
	if err != nil {
		t.Fatalf("MarshalJSON: %v", err)
	}
	if len(data) == 0 {
		t.Fatal("expected non-empty completion data")
	}
}

func TestBindFlagCompletionsSetsAnnotation(t *testing.T) {
	cmd := &cobra.Command{Use: "test"}
	cmd.Flags().String("target", "", "target host")

	BindFlagCompletions(cmd, func(comp carapace.ActionMap) {
		comp["target"] = carapace.ActionValues("host-a", "host-b")
	})

	flag := cmd.Flags().Lookup("target")
	if flag == nil {
		t.Fatal("target flag not found")
	}
	vals, ok := flag.Annotations["ui:hasCompletion"]
	if !ok || len(vals) == 0 || vals[0] != "true" {
		t.Fatalf("expected ui:hasCompletion annotation, got %v", flag.Annotations)
	}

	if !HasFlagCompletions(cmd) {
		t.Fatal("HasFlagCompletions should return true")
	}
}
