package core

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/chainreactors/IoM-go/consts"
	"github.com/chainreactors/IoM-go/proto/implant/implantpb"
	"google.golang.org/protobuf/encoding/protojson"
)

func TestBuildTaskRequestSummaryExecuteBinaryDemo(t *testing.T) {
	spite := &implantpb.Spite{
		Name:    consts.ModuleExecuteExe,
		TaskId:  42,
		Async:   true,
		Timeout: 60,
		Body: &implantpb.Spite_ExecuteBinary{
			ExecuteBinary: &implantpb.ExecuteBinary{
				Name:    "whoami.exe",
				Bin:     []byte("fake-pe-binary"),
				Type:    consts.ModuleExecuteExe,
				Args:    []string{"--help"},
				Output:  true,
				Timeout: 60,
			},
		},
	}

	summary := BuildTaskRequestSummary(spite)
	if summary.Command != "execute_exe whoami.exe -- --help" {
		t.Fatalf("command summary = %q", summary.Command)
	}
	if len(summary.Payloads) != 1 {
		t.Fatalf("payloads = %d, want 1", len(summary.Payloads))
	}
	if summary.Payloads[0].Path != "execute_binary.bin" {
		t.Fatalf("payload path = %q", summary.Payloads[0].Path)
	}
	if summary.Payloads[0].Size != len("fake-pe-binary") {
		t.Fatalf("payload size = %d", summary.Payloads[0].Size)
	}

	summaryJSON, err := json.MarshalIndent(summary, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	requestJSON, err := protojson.MarshalOptions{
		Multiline: true,
		Indent:    "  ",
	}.Marshal(spite)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(requestJSON), "ZmFrZS1wZS1iaW5hcnk=") {
		t.Fatalf("raw request does not contain base64 encoded bin: %s", requestJSON)
	}

	t.Logf("summary:\n%s", summaryJSON)
	t.Logf("raw request:\n%s", requestJSON)
}
