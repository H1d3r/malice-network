package common_test

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/carapace-sh/carapace"
	"github.com/chainreactors/IoM-go/proto/client/clientpb"
	"github.com/chainreactors/malice-network/client/command/common"
	"github.com/chainreactors/malice-network/client/command/testsupport"
)

func TestForwardListenerIDCompleterUsesForwardListenersOnly(t *testing.T) {
	h := testsupport.NewClientHarness(t)
	h.Console.Listeners["listener-a"] = &clientpb.Listener{Id: "listener-a"}
	h.Console.Listeners["listener-normal"] = &clientpb.Listener{Id: "listener-normal"}
	h.Recorder.OnForwardListenerStatuses("ListForwardListeners", func(_ context.Context, _ any) (*clientpb.ForwardListenerStatuses, error) {
		return &clientpb.ForwardListenerStatuses{Listeners: []*clientpb.ForwardListenerStatus{
			{ListenerId: "listener-a", Address: "127.0.0.1:5005", Active: true},
		}}, nil
	})

	values := completionValues(t, common.ForwardListenerIDCompleter(h.Console))

	if !hasCompletionValue(values, "listener-a") {
		t.Fatalf("completion values = %#v, want listener-a", values)
	}
	if hasCompletionValue(values, "listener-normal") {
		t.Fatalf("completion values = %#v, should not include normal listeners", values)
	}
}

func completionValues(t testing.TB, action carapace.Action) []string {
	t.Helper()

	data, err := json.Marshal(action.Invoke(carapace.Context{}))
	if err != nil {
		t.Fatalf("marshal completion action failed: %v", err)
	}
	var decoded struct {
		Values []struct {
			Value string `json:"value"`
		} `json:"values"`
	}
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("decode completion action failed: %v", err)
	}
	values := make([]string, 0, len(decoded.Values))
	for _, value := range decoded.Values {
		values = append(values, value.Value)
	}
	return values
}

func hasCompletionValue(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}
