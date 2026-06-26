package common_test

import (
	"testing"

	"github.com/chainreactors/IoM-go/consts"
	"github.com/chainreactors/IoM-go/proto/client/clientpb"
	"github.com/chainreactors/malice-network/client/command/common"
	"github.com/chainreactors/malice-network/client/command/testsupport"
)

func TestSessionTaskCompleterUsesCurrentSessionTasks(t *testing.T) {
	h := testsupport.NewHarness(t)
	h.Session.Tasks = &clientpb.Tasks{Tasks: []*clientpb.Task{
		{TaskId: 7, Type: consts.ModuleWhoami, Cur: 1, Total: 1},
		{TaskId: 8, Type: consts.ModuleLs, Cur: 0, Total: 1},
		nil,
	}}

	values := completionValues(t, common.SessionTaskCompleter(h.Console))

	if !hasCompletionValue(values, "7") {
		t.Fatalf("completion values = %#v, want task 7", values)
	}
	if !hasCompletionValue(values, "8") {
		t.Fatalf("completion values = %#v, want task 8", values)
	}
}
