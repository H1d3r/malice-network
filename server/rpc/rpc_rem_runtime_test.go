package rpc

import (
	"context"
	"strings"
	"testing"

	"github.com/chainreactors/IoM-go/consts"
	"github.com/chainreactors/IoM-go/proto/client/clientpb"
	"github.com/chainreactors/IoM-go/types"
	"github.com/chainreactors/malice-network/helper/utils/configutil"
	"github.com/chainreactors/malice-network/server/internal/configs"
	"github.com/chainreactors/malice-network/server/internal/core"
	"github.com/chainreactors/malice-network/server/internal/db"
	"github.com/chainreactors/malice-network/server/internal/db/models"
)

func TestRemAgentCtrlPropagatesListenerFailure(t *testing.T) {
	newRPCTestEnv(t)
	listener, pipeline := seedRemRuntime(t, "rem-runtime-failure")

	go func() {
		ctrl := <-listener.Ctrl
		listener.CtrlJob.Store(ctrl.Id, &clientpb.JobStatus{
			CtrlId: ctrl.Id,
			Status: consts.CtrlStatusFailed,
			Error:  "listener ctrl failed",
		})
	}()

	_, err := (&Server{}).RemAgentCtrl(context.Background(), &clientpb.REMAgent{
		PipelineId: pipeline.Name,
		Id:         "agent-1",
	})
	if err == nil || !strings.Contains(err.Error(), "listener ctrl failed") {
		t.Fatalf("RemAgentCtrl error = %v, want listener failure", err)
	}
}

func TestRemAgentLogRejectsMissingLogPayload(t *testing.T) {
	newRPCTestEnv(t)
	listener, pipeline := seedRemRuntime(t, "rem-runtime-log")

	go func() {
		ctrl := <-listener.Ctrl
		listener.CtrlJob.Store(ctrl.Id, &clientpb.JobStatus{
			CtrlId: ctrl.Id,
			Status: consts.CtrlStatusSuccess,
		})
	}()

	_, err := (&Server{}).RemAgentLog(context.Background(), &clientpb.REMAgent{
		PipelineId: pipeline.Name,
		Id:         "agent-2",
	})
	if err == nil || !strings.Contains(err.Error(), "missing log") {
		t.Fatalf("RemAgentLog error = %v, want missing log error", err)
	}
}

func TestRemAgentHandlersRejectNilRequest(t *testing.T) {
	if _, err := (&Server{}).RemAgentCtrl(context.Background(), nil); err == nil || !strings.Contains(err.Error(), types.ErrMissingRequestField.Error()) {
		t.Fatalf("RemAgentCtrl(nil) error = %v, want %v", err, types.ErrMissingRequestField)
	}
	if _, err := (&Server{}).RemAgentLog(context.Background(), nil); err == nil || !strings.Contains(err.Error(), types.ErrMissingRequestField.Error()) {
		t.Fatalf("RemAgentLog(nil) error = %v, want %v", err, types.ErrMissingRequestField)
	}
}

func TestRegisterRemPreservesExistingDBAndBackfillsConfig(t *testing.T) {
	newRPCTestEnv(t)
	listener := core.NewListener("listener-rem-preserve", "127.0.0.1")
	core.Listeners.Add(listener)
	if err := configutil.SetStructByTag("listeners", &configs.ListenerConfig{
		Name: "listener-rem-preserve",
		REMs: []*configs.REMConfig{
			{Enable: true, Name: "rem-preserve", Console: "tcp://0.0.0.0:20001"},
		},
	}, "config"); err != nil {
		t.Fatalf("SetStructByTag failed: %v", err)
	}
	existing := &clientpb.Pipeline{
		Name:       "rem-preserve",
		ListenerId: listener.Name,
		Type:       consts.RemPipeline,
		Body: &clientpb.Pipeline_Rem{
			Rem: &clientpb.REM{
				Name:       "rem-preserve",
				ListenerId: listener.Name,
				Console:    "tcp://0.0.0.0:20000",
				Link:       "tcp://127.0.0.1:20000",
			},
		},
	}
	if _, err := db.SavePipeline(models.FromPipelinePb(existing)); err != nil {
		t.Fatalf("SavePipeline failed: %v", err)
	}

	_, err := (&Server{}).RegisterRem(context.Background(), &clientpb.Pipeline{
		Name:       "rem-preserve",
		ListenerId: listener.Name,
		Type:       consts.RemPipeline,
		Body: &clientpb.Pipeline_Rem{
			Rem: &clientpb.REM{
				Console: "tcp://0.0.0.0:29999",
			},
		},
	})
	if err != nil {
		t.Fatalf("RegisterRem failed: %v", err)
	}

	found, err := db.FindPipelineByListener("rem-preserve", listener.Name)
	if err != nil {
		t.Fatalf("FindPipelineByListener failed: %v", err)
	}
	if found.Console != "tcp://0.0.0.0:20000" || found.PipelineParams.Link != "tcp://127.0.0.1:20000" {
		t.Fatalf("DB REM was overwritten: console=%q link=%q", found.Console, found.PipelineParams.Link)
	}
	cfg := configs.GetListenerConfig()
	if cfg.REMs[0].Console != "tcp://0.0.0.0:20000" || cfg.REMs[0].Link != "tcp://127.0.0.1:20000" {
		t.Fatalf("config REM was not backfilled from DB: %#v", cfg.REMs[0])
	}
}

func TestSyncPipelineWritesRuntimeRemLinkToConfig(t *testing.T) {
	newRPCTestEnv(t)
	if err := configutil.SetStructByTag("listeners", &configs.ListenerConfig{
		Name: "listener-sync-rem",
		REMs: []*configs.REMConfig{
			{Enable: true, Name: "rem-sync", Console: "tcp://0.0.0.0:21000"},
		},
	}, "config"); err != nil {
		t.Fatalf("SetStructByTag failed: %v", err)
	}
	pipeline := &clientpb.Pipeline{
		Name:       "rem-sync",
		ListenerId: "listener-sync-rem",
		Type:       consts.RemPipeline,
		Body: &clientpb.Pipeline_Rem{
			Rem: &clientpb.REM{
				Name:       "rem-sync",
				ListenerId: "listener-sync-rem",
				Console:    "tcp://0.0.0.0:21000",
				Link:       "tcp://127.0.0.1:21000",
			},
		},
	}

	if _, err := (&Server{}).SyncPipeline(context.Background(), pipeline); err != nil {
		t.Fatalf("SyncPipeline failed: %v", err)
	}

	cfg := configs.GetListenerConfig()
	if cfg.REMs[0].Link != "tcp://127.0.0.1:21000" {
		t.Fatalf("runtime link was not synced to config: %#v", cfg.REMs[0])
	}
}

func seedRemRuntime(t testing.TB, name string) (*core.Listener, *clientpb.Pipeline) {
	t.Helper()

	listener := core.NewListener("listener-"+name, "127.0.0.1")
	core.Listeners.Add(listener)
	pipeline := &clientpb.Pipeline{
		Name:       name,
		ListenerId: listener.Name,
		Type:       consts.RemPipeline,
		Body: &clientpb.Pipeline_Rem{
			Rem: &clientpb.REM{
				Agents: map[string]*clientpb.REMAgent{},
			},
		},
	}
	listener.AddPipeline(pipeline)
	return listener, pipeline
}
