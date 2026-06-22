package listener

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/chainreactors/IoM-go/consts"
	"github.com/chainreactors/IoM-go/proto/client/clientpb"
	"github.com/chainreactors/malice-network/server/internal/configs"
	"github.com/chainreactors/malice-network/server/internal/core"
)

func TestHandleRetireDeletesRequestedFilesAndSchedulesShutdown(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "listener.yaml")
	authPath := filepath.Join(dir, "listener.auth")
	if err := os.WriteFile(configPath, []byte("listeners: {}\n"), 0600); err != nil {
		t.Fatalf("write config: %v", err)
	}
	if err := os.WriteFile(authPath, []byte("auth"), 0600); err != nil {
		t.Fatalf("write auth: %v", err)
	}

	oldConfigPath := configs.CurrentServerConfigFilename
	configs.CurrentServerConfigFilename = configPath
	t.Cleanup(func() { configs.CurrentServerConfigFilename = oldConfigPath })

	shutdown := make(chan struct{})
	lns := &listener{
		Name:      "listener-a",
		cfg:       &configs.ListenerConfig{Auth: authPath},
		pipelines: core.NewPipelines(),
		websites:  map[string]*Website{},
		shutdown: func() error {
			close(shutdown)
			return nil
		},
	}

	status := lns.handleJobCtrl(&clientpb.JobCtrl{
		Id:   12,
		Ctrl: consts.CtrlListenerRetire,
		Retire: &clientpb.ListenerRetire{
			ListenerId:  "listener-a",
			PurgeConfig: true,
			PurgeAuth:   true,
		},
	})
	if status == nil || status.Status != consts.CtrlStatusSuccess {
		t.Fatalf("status = %#v, want success", status)
	}
	if _, err := os.Stat(configPath); !os.IsNotExist(err) {
		t.Fatalf("config stat error = %v, want not exist", err)
	}
	if _, err := os.Stat(authPath); !os.IsNotExist(err) {
		t.Fatalf("auth stat error = %v, want not exist", err)
	}

	select {
	case <-shutdown:
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for listener shutdown")
	}
}
