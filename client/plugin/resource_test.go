package plugin

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/chainreactors/malice-network/client/assets"
	lua "github.com/yuin/gopher-lua"
)

func newResourceTestPlugin(t *testing.T) (*LuaPlugin, *lua.LState) {
	t.Helper()

	oldRoot := assets.MaliceDirName
	assets.MaliceDirName = t.TempDir()
	t.Cleanup(func() {
		assets.MaliceDirName = oldRoot
	})

	plug := &LuaPlugin{
		DefaultPlugin: &DefaultPlugin{
			MalManiFest: &MalManiFest{Name: "resource-test"},
			CMDs:        make(Commands),
		},
	}
	plug.RegisterLuaFunction()

	vm := lua.NewState()
	t.Cleanup(vm.Close)
	if err := plug.InitLuaContext(vm); err != nil {
		t.Fatalf("InitLuaContext failed: %v", err)
	}

	return plug, vm
}

func TestLuaPluginListResourceListsSortedEntries(t *testing.T) {
	_, vm := newResourceTestPlugin(t)

	modulesDir := filepath.Join(assets.GetMalsDir(), "resource-test", "resources", "modules")
	if err := os.MkdirAll(filepath.Join(modulesDir, "nested"), 0o700); err != nil {
		t.Fatalf("create nested resource dir: %v", err)
	}
	for _, name := range []string{"z.dll", "a.dll"} {
		if err := os.WriteFile(filepath.Join(modulesDir, name), []byte("test"), 0o600); err != nil {
			t.Fatalf("write resource %s: %v", name, err)
		}
	}

	if err := vm.DoString(`
entries = list_resource("modules")
result = table.concat(entries, ",")
`); err != nil {
		t.Fatalf("list_resource failed: %v", err)
	}

	if got, want := vm.GetGlobal("result").String(), "a.dll,nested,z.dll"; got != want {
		t.Fatalf("list_resource returned %q, want %q", got, want)
	}
}

func TestLuaPluginListResourceRejectsEscapingPath(t *testing.T) {
	_, vm := newResourceTestPlugin(t)

	if err := vm.DoString(`
ok, err = pcall(function()
  return list_resource("../outside")
end)
`); err != nil {
		t.Fatalf("pcall failed: %v", err)
	}

	if got := vm.GetGlobal("ok"); got != lua.LFalse {
		t.Fatalf("list_resource escape unexpectedly succeeded: %s", got.String())
	}
	if errText := vm.GetGlobal("err").String(); !strings.Contains(errText, "invalid resource directory") {
		t.Fatalf("list_resource escape error = %q, want invalid resource directory", errText)
	}
}

func TestLuaPluginListResourceRejectsParentSegments(t *testing.T) {
	_, vm := newResourceTestPlugin(t)

	if err := vm.DoString(`
ok, err = pcall(function()
  return list_resource("modules/../outside")
end)
`); err != nil {
		t.Fatalf("pcall failed: %v", err)
	}

	if got := vm.GetGlobal("ok"); got != lua.LFalse {
		t.Fatalf("list_resource parent segment unexpectedly succeeded: %s", got.String())
	}
	if errText := vm.GetGlobal("err").String(); !strings.Contains(errText, "invalid resource directory") {
		t.Fatalf("list_resource parent segment error = %q, want invalid resource directory", errText)
	}
}
