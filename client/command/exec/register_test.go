package exec

import (
	"testing"

	"github.com/chainreactors/IoM-go/consts"
	"github.com/chainreactors/malice-network/client/core"
	"github.com/chainreactors/malice-network/helper/intermediate"
	"github.com/spf13/cobra"
)

func TestRegisterExecuteFuncShellHelperMatchesLuaSignature(t *testing.T) {
	restore := preserveInternalFunctions(
		consts.ModuleExecute,
		consts.ModuleAliasRun,
		consts.ModuleAliasExecute,
		consts.ModuleAliasShell,
		"bshell",
	)
	defer restore()

	con := &core.Console{
		CMDs:    map[string]*cobra.Command{},
		Helpers: map[string]*cobra.Command{},
	}
	RegisterExecuteFunc(con)

	fn := intermediate.InternalFunctions[consts.ModuleAliasShell]
	if fn == nil {
		t.Fatalf("%s was not registered", consts.ModuleAliasShell)
	}
	if got, want := len(fn.ArgTypes), 3; got != want {
		t.Fatalf("shell ArgTypes len = %d, want %d", got, want)
	}
	if fn.Helper == nil {
		t.Fatal("shell helper was not registered")
	}
	if got, want := len(fn.Helper.Input), len(fn.ArgTypes); got != want {
		t.Fatalf("shell helper input len = %d, want ArgTypes len %d", got, want)
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
