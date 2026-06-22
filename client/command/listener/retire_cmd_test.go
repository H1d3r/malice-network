package listener_test

import (
	"context"
	"testing"

	"github.com/chainreactors/IoM-go/proto/client/clientpb"
	"github.com/chainreactors/malice-network/client/command/testsupport"
)

func TestListenerRetireCommandSendsTypedRPC(t *testing.T) {
	h := testsupport.NewClientHarness(t)
	h.Recorder.OnForwardListenerStatus("RetireListener", func(_ context.Context, _ any) (*clientpb.ForwardListenerStatus, error) {
		return &clientpb.ForwardListenerStatus{ListenerId: "listener-a"}, nil
	})

	if err := h.ExecuteClient("listener", "retire", "listener-a", "--purge-config", "--purge-auth", "--no-revoke", "--timeout", "7", "--yes"); err != nil {
		t.Fatalf("listener retire failed: %v", err)
	}

	req, _ := testsupport.MustSingleCall[*clientpb.ListenerRetire](t, h, "RetireListener")
	if req.ListenerId != "listener-a" || !req.PurgeConfig || !req.PurgeAuth || !req.NoRevoke || req.TimeoutSeconds != 7 {
		t.Fatalf("request = %#v, want listener-a purge config/auth no-revoke timeout=7", req)
	}
}
