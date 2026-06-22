package server

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/chainreactors/malice-network/server/internal/configs"
)

func TestValidateAllowsListenerOnlyWithoutServerConfig(t *testing.T) {
	opt := &Options{
		ListenerOnly: true,
		Listeners: &configs.ListenerConfig{
			Enable: true,
		},
	}

	if err := opt.Validate(); err != nil {
		t.Fatalf("Validate() error = %v, want nil", err)
	}
}

func TestPrepareListenerOnlyDoesNotRequireServerConfig(t *testing.T) {
	opt := &Options{
		ListenerOnly: true,
		Listeners: &configs.ListenerConfig{
			Enable: true,
		},
	}

	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("PrepareListener() panicked with nil server config: %v", r)
		}
	}()

	if err := opt.PrepareListener(); err == nil {
		t.Fatal("PrepareListener() error = nil, want listener startup error")
	}
}

func TestPrepareConfigAllowsListenerOnlyFileWithoutServerSection(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), "listener.yaml")
	content := []byte(`listeners:
  enable: true
  name: listener-a
  auth: listener-a.auth
  transport: forward
  forward:
    listen_host: 0.0.0.0
    listen_port: 5005
`)
	if err := os.WriteFile(configPath, content, 0600); err != nil {
		t.Fatalf("write config fixture: %v", err)
	}

	opt := &Options{
		Config:       configPath,
		ListenerOnly: true,
	}
	if err := opt.PrepareConfig(nil); err != nil {
		t.Fatalf("PrepareConfig() error = %v, want nil", err)
	}
	if opt.Listeners == nil {
		t.Fatal("PrepareConfig() did not load listeners config")
	}
	if opt.Server != nil {
		t.Fatalf("PrepareConfig() server config = %#v, want nil", opt.Server)
	}
}
