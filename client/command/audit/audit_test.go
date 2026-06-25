package audit_test

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/chainreactors/IoM-go/consts"
	"github.com/chainreactors/IoM-go/proto/client/clientpb"
	implantpb "github.com/chainreactors/IoM-go/proto/implant/implantpb"
	"github.com/chainreactors/malice-network/client/assets"
	"github.com/chainreactors/malice-network/client/command/testsupport"
	"github.com/chainreactors/malice-network/helper/intermediate"
)

func TestAuditCommandConformance(t *testing.T) {
	testsupport.RunClientCases(t, []testsupport.CommandCase{
		{
			Name: "audit session exports enriched json",
			Argv: []string{consts.CommandAudit, consts.CommandSession, "audit-session-1", "-o", "json"},
			Setup: func(t testing.TB, h *testsupport.Harness) {
				registerAuditFixtureCallback(t, "audit_test")
				h.Recorder.OnAudits("GetAudit", func(ctx context.Context, request any) (*clientpb.Audits, error) {
					return auditFixture("audit-session-1"), nil
				})
			},
			Assert: func(t testing.TB, h *testsupport.Harness, err error) {
				req, _ := testsupport.MustSingleCall[*clientpb.SessionRequest](t, h, "GetAudit")
				if req.SessionId != "audit-session-1" {
					t.Fatalf("audit session id = %q, want audit-session-1", req.SessionId)
				}

				exports := readAuditExport(t, "audit-session-1")
				if len(exports) != 1 {
					t.Fatalf("audit entries = %d, want 1", len(exports))
				}
				entry := exports[0]
				assertJSONValue(t, entry, "session", "audit-session-1")
				assertJSONValue(t, entry, "task", "7")
				assertJSONValue(t, entry, "type", "audit_test")
				assertJSONValue(t, entry, "callby", "operator-a")
				assertJSONValue(t, entry, "taskFinished", true)
				assertJSONValue(t, entry, "timeout", false)
				assertJSONValue(t, entry, "createdAt", float64(1710000000))
				assertJSONValue(t, entry, "finishedAt", float64(1710000060))
				assertJSONValue(t, entry, "commandSummary", "audit_test --value alpha")
				assertJSONValue(t, entry, "requestSummary", `{"value":"alpha"}`)
				assertJSONValue(t, entry, "requestSize", float64(64))
				assertJSONValue(t, entry, "requestSha256", "sha256-audit-test")
				assertJSONValue(t, entry, "hasRequest", true)
				assertJSONValue(t, entry, "resultIndex", float64(3))
				assertJSONValue(t, entry, "taskResult", "rendered audit result")
			},
		},
		{
			Name: "audit session id shorthand exports audit",
			Argv: []string{consts.CommandAudit, "audit-session-2", "-o", "json"},
			Setup: func(t testing.TB, h *testsupport.Harness) {
				h.Recorder.OnAudits("GetAudit", func(ctx context.Context, request any) (*clientpb.Audits, error) {
					return &clientpb.Audits{}, nil
				})
			},
			Assert: func(t testing.TB, h *testsupport.Harness, err error) {
				req, _ := testsupport.MustSingleCall[*clientpb.SessionRequest](t, h, "GetAudit")
				if req.SessionId != "audit-session-2" {
					t.Fatalf("audit shorthand session id = %q, want audit-session-2", req.SessionId)
				}
				_ = readAuditExport(t, "audit-session-2")
			},
		},
		{
			Name:    "audit session rejects missing session id",
			Argv:    []string{consts.CommandAudit, consts.CommandSession},
			WantErr: "session id is required",
			Assert: func(t testing.TB, h *testsupport.Harness, err error) {
				testsupport.RequireNoPrimaryCalls(t, h)
			},
		},
	})
}

func registerAuditFixtureCallback(t testing.TB, name string) {
	t.Helper()

	original, hadOriginal := intermediate.InternalFunctions[name]
	intermediate.InternalFunctions[name] = &intermediate.InternalFunc{
		FinishCallback: func(content *clientpb.TaskContext) (string, error) {
			return "rendered audit result", nil
		},
	}
	t.Cleanup(func() {
		if hadOriginal {
			intermediate.InternalFunctions[name] = original
			return
		}
		delete(intermediate.InternalFunctions, name)
	})
}

func auditFixture(sessionID string) *clientpb.Audits {
	return &clientpb.Audits{
		Audit: []*clientpb.Audit{
			{
				Context: &clientpb.TaskContext{
					Session: &clientpb.Session{SessionId: sessionID},
					Task: &clientpb.Task{
						TaskId:         7,
						SessionId:      sessionID,
						Type:           "audit_test",
						Cur:            1,
						Total:          1,
						Finished:       true,
						Timeout:        false,
						CreatedAt:      1710000000,
						FinishedAt:     1710000060,
						CommandSummary: "audit_test --value alpha",
						RequestSummary: `{"value":"alpha"}`,
						RequestSize:    64,
						RequestSha256:  "sha256-audit-test",
						HasRequest:     true,
						Callby:         "operator-a",
					},
					Spite: &implantpb.Spite{
						TaskId: 7,
						Name:   "audit_test",
						Body:   &implantpb.Spite_Empty{Empty: &implantpb.Empty{}},
					},
				},
				Command:     "legacy command",
				Request:     &implantpb.Spite{TaskId: 7, Name: "audit_test"},
				Created:     "2024-03-09 16:00:00",
				Finished:    "2024-03-09 16:01:00",
				Lasted:      "2024-03-09 16:01:00",
				ResultIndex: 3,
			},
		},
	}
}

func readAuditExport(t testing.TB, sessionID string) []map[string]any {
	t.Helper()

	path := filepath.Join(assets.GetTempDir(), sessionID+".json")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read audit export %s: %v", path, err)
	}
	t.Cleanup(func() {
		_ = os.Remove(path)
	})

	var entries []map[string]any
	if err := json.Unmarshal(data, &entries); err != nil {
		t.Fatalf("decode audit export: %v\n%s", err, string(data))
	}
	return entries
}

func assertJSONValue(t testing.TB, entry map[string]any, key string, want any) {
	t.Helper()

	got, ok := entry[key]
	if !ok {
		t.Fatalf("missing json key %q in %#v", key, entry)
	}
	if got != want {
		t.Fatalf("json key %q = %#v (%T), want %#v (%T)", key, got, got, want, want)
	}
}
