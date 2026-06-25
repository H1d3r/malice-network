package rpc

import (
	"context"
	"testing"

	"github.com/chainreactors/IoM-go/consts"
	"github.com/chainreactors/IoM-go/proto/client/clientpb"
	implantpb "github.com/chainreactors/IoM-go/proto/implant/implantpb"
	"github.com/chainreactors/malice-network/server/internal/db"
	"github.com/gookit/config/v2"
)

func TestGetAuditIncludesResultIndexAndSortedTaskResults(t *testing.T) {
	env := newRPCTestEnv(t)
	sess := env.seedSession(t, "rpc-audit-results", "rpc-audit-results-pipe", true)
	originalAuditLevel := config.Int(consts.ConfigAuditLevel)
	config.Set(consts.ConfigAuditLevel, 2)
	t.Cleanup(func() {
		config.Set(consts.ConfigAuditLevel, originalAuditLevel)
	})

	for _, taskID := range []uint32{2, 10} {
		if err := db.AddTask(&clientpb.Task{
			SessionId: sess.ID,
			TaskId:    taskID,
			Type:      "audit",
			Cur:       1,
			Total:     1,
		}); err != nil {
			t.Fatalf("AddTask(%d) failed: %v", taskID, err)
		}
	}

	writeHistoryTaskSpite(t, sess.ID, 10, 0, &implantpb.Spite{
		TaskId: 10,
		Body:   &implantpb.Spite_Ping{Ping: &implantpb.Ping{Nonce: 100}},
	})
	writeHistoryTaskSpite(t, sess.ID, 2, 1, &implantpb.Spite{
		TaskId: 2,
		Body:   &implantpb.Spite_Ping{Ping: &implantpb.Ping{Nonce: 21}},
	})
	writeHistoryTaskSpite(t, sess.ID, 2, 0, &implantpb.Spite{
		TaskId: 2,
		Body:   &implantpb.Spite_Ping{Ping: &implantpb.Ping{Nonce: 20}},
	})

	audits, err := (&Server{}).GetAudit(context.Background(), &clientpb.SessionRequest{SessionId: sess.ID})
	if err != nil {
		t.Fatalf("GetAudit failed: %v", err)
	}

	if len(audits.GetAudit()) != 3 {
		t.Fatalf("audit count = %d, want 3", len(audits.GetAudit()))
	}
	assertAuditEntry(t, audits.GetAudit()[0], 2, 0, 20)
	assertAuditEntry(t, audits.GetAudit()[1], 2, 1, 21)
	assertAuditEntry(t, audits.GetAudit()[2], 10, 0, 100)
	for i, entry := range audits.GetAudit() {
		if entry.GetRequest() != nil {
			t.Fatalf("audit entry %d request = %#v, want nil request with response preserved", i, entry.GetRequest())
		}
	}
}

func assertAuditEntry(t testing.TB, entry *clientpb.Audit, wantTask uint32, wantIndex int32, wantNonce int32) {
	t.Helper()

	if entry.GetContext().GetTask().GetTaskId() != wantTask {
		t.Fatalf("audit task id = %d, want %d", entry.GetContext().GetTask().GetTaskId(), wantTask)
	}
	if entry.GetResultIndex() != wantIndex {
		t.Fatalf("audit result index = %d, want %d", entry.GetResultIndex(), wantIndex)
	}
	if entry.GetContext().GetSpite().GetPing().GetNonce() != wantNonce {
		t.Fatalf("audit nonce = %d, want %d", entry.GetContext().GetSpite().GetPing().GetNonce(), wantNonce)
	}
}
