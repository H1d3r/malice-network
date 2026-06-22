package agent

import (
	"testing"

	"github.com/chainreactors/malice-network/client/core"
	"github.com/chainreactors/malice-network/helper/intermediate"
	"github.com/spf13/cobra"
)

func TestRegisterToolCallFuncHelperMatchesLuaSignature(t *testing.T) {
	restore := preserveInternalFunctions(ModuleToolInject)
	defer restore()

	con := &core.Console{
		CMDs:    map[string]*cobra.Command{},
		Helpers: map[string]*cobra.Command{},
	}
	RegisterToolCallFunc(con)

	fn := intermediate.InternalFunctions[ModuleToolInject]
	if fn == nil {
		t.Fatalf("%s was not registered", ModuleToolInject)
	}
	if got, want := len(fn.ArgTypes), 3; got != want {
		t.Fatalf("tool_inject ArgTypes len = %d, want %d", got, want)
	}
	if fn.Helper == nil {
		t.Fatal("tool_inject helper was not registered")
	}
	if got, want := len(fn.Helper.Input), len(fn.ArgTypes); got != want {
		t.Fatalf("tool_inject helper input len = %d, want ArgTypes len %d", got, want)
	}
}

func preserveInternalFunctions(names ...string) func() {
	saved := make(map[string]*intermediate.InternalFunc, len(names))
	existed := make(map[string]bool, len(names))
	for _, name := range names {
		if fn, ok := intermediate.InternalFunctions[name]; ok {
			saved[name] = fn
			existed[name] = true
		}
		delete(intermediate.InternalFunctions, name)
	}

	return func() {
		for _, name := range names {
			if existed[name] {
				intermediate.InternalFunctions[name] = saved[name]
			} else {
				delete(intermediate.InternalFunctions, name)
			}
		}
	}
}
