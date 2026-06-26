package listener_test

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/chainreactors/IoM-go/consts"
	"github.com/chainreactors/IoM-go/proto/client/clientpb"
	listenercmd "github.com/chainreactors/malice-network/client/command/listener"
	"github.com/chainreactors/malice-network/client/command/testsupport"
	"github.com/spf13/cobra"
)

func TestStartPipelineCmdPropagatesStopFailure(t *testing.T) {
	h := testsupport.NewClientHarness(t)
	h.Console.Pipelines["pipe-a"] = &clientpb.Pipeline{Name: "pipe-a", Enable: true}
	h.Recorder.OnEmpty("StopPipeline", func(context.Context, any) (*clientpb.Empty, error) {
		return nil, errors.New("stop failed")
	})

	cmd := &cobra.Command{Use: "start"}
	cmd.Flags().String("cert-name", "", "")
	if err := cmd.Flags().Parse([]string{"pipe-a"}); err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	err := listenercmd.StartPipelineCmd(cmd, h.Console)
	if err == nil || !strings.Contains(err.Error(), "stop failed") {
		t.Fatalf("StartPipelineCmd error = %v, want stop failure", err)
	}

	calls := h.Recorder.Calls()
	if len(calls) != 1 || calls[0].Method != "StopPipeline" {
		t.Fatalf("calls = %#v, want only StopPipeline", calls)
	}
}

func TestStartPipelineCmdForwardsCertNameToStartRequest(t *testing.T) {
	h := testsupport.NewClientHarness(t)
	h.Console.Pipelines["pipe-b"] = &clientpb.Pipeline{Name: "pipe-b", Enable: false}

	cmd := &cobra.Command{Use: "start"}
	cmd.Flags().String("cert-name", "", "")
	if err := cmd.Flags().Parse([]string{"pipe-b"}); err != nil {
		t.Fatalf("Parse failed: %v", err)
	}
	if err := cmd.Flags().Set("cert-name", "cert-blue"); err != nil {
		t.Fatalf("Set(cert-name) failed: %v", err)
	}

	if err := listenercmd.StartPipelineCmd(cmd, h.Console); err != nil {
		t.Fatalf("StartPipelineCmd failed: %v", err)
	}

	calls := h.Recorder.Calls()
	if len(calls) != 1 || calls[0].Method != "StartPipeline" {
		t.Fatalf("calls = %#v, want only StartPipeline", calls)
	}
	req, ok := calls[0].Request.(*clientpb.CtrlPipeline)
	if !ok {
		t.Fatalf("request type = %T, want *clientpb.CtrlPipeline", calls[0].Request)
	}
	if req.Name != "pipe-b" || req.CertName != "cert-blue" {
		t.Fatalf("start request = %#v, want pipe-b/cert-blue", req)
	}
}

func TestStartPipelineCmdUsesScopedCacheKey(t *testing.T) {
	h := testsupport.NewClientHarness(t)
	h.Console.Pipelines["listener-a:pipe-c"] = &clientpb.Pipeline{
		Name:       "pipe-c",
		ListenerId: "listener-a",
		Enable:     false,
	}

	cmd := &cobra.Command{Use: "start"}
	cmd.Flags().String("cert-name", "", "")
	if err := cmd.Flags().Parse([]string{"listener-a:pipe-c"}); err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if err := listenercmd.StartPipelineCmd(cmd, h.Console); err != nil {
		t.Fatalf("StartPipelineCmd failed: %v", err)
	}

	calls := h.Recorder.Calls()
	if len(calls) != 1 || calls[0].Method != "StartPipeline" {
		t.Fatalf("calls = %#v, want only StartPipeline", calls)
	}
	req, ok := calls[0].Request.(*clientpb.CtrlPipeline)
	if !ok {
		t.Fatalf("request type = %T, want *clientpb.CtrlPipeline", calls[0].Request)
	}
	if req.Name != "pipe-c" || req.ListenerId != "listener-a" {
		t.Fatalf("start request = %#v, want scoped pipe-c/listener-a", req)
	}
}

func TestStartPipelineCmdParsesScopedNameWithoutCache(t *testing.T) {
	h := testsupport.NewClientHarness(t)

	cmd := &cobra.Command{Use: "start"}
	cmd.Flags().String("cert-name", "", "")
	if err := cmd.Flags().Parse([]string{"listener-a:pipe-c"}); err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if err := listenercmd.StartPipelineCmd(cmd, h.Console); err != nil {
		t.Fatalf("StartPipelineCmd failed: %v", err)
	}

	calls := h.Recorder.Calls()
	if len(calls) != 1 || calls[0].Method != "StartPipeline" {
		t.Fatalf("calls = %#v, want only StartPipeline", calls)
	}
	req, ok := calls[0].Request.(*clientpb.CtrlPipeline)
	if !ok {
		t.Fatalf("request type = %T, want *clientpb.CtrlPipeline", calls[0].Request)
	}
	if req.Name != "pipe-c" || req.ListenerId != "listener-a" {
		t.Fatalf("start request = %#v, want scoped pipe-c/listener-a", req)
	}
}

func TestStopPipelineCmdUsesScopedCacheKey(t *testing.T) {
	h := testsupport.NewClientHarness(t)
	h.Console.Pipelines["listener-b:pipe-d"] = &clientpb.Pipeline{
		Name:       "pipe-d",
		ListenerId: "listener-b",
	}

	cmd := &cobra.Command{Use: "stop"}
	if err := cmd.Flags().Parse([]string{"listener-b:pipe-d"}); err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if err := listenercmd.StopPipelineCmd(cmd, h.Console); err != nil {
		t.Fatalf("StopPipelineCmd failed: %v", err)
	}

	calls := h.Recorder.Calls()
	if len(calls) != 1 || calls[0].Method != "StopPipeline" {
		t.Fatalf("calls = %#v, want only StopPipeline", calls)
	}
	req, ok := calls[0].Request.(*clientpb.CtrlPipeline)
	if !ok {
		t.Fatalf("request type = %T, want *clientpb.CtrlPipeline", calls[0].Request)
	}
	if req.Name != "pipe-d" || req.ListenerId != "listener-b" {
		t.Fatalf("stop request = %#v, want scoped pipe-d/listener-b", req)
	}
}

func TestStopPipelineCmdParsesScopedNameWithoutCache(t *testing.T) {
	h := testsupport.NewClientHarness(t)

	cmd := &cobra.Command{Use: "stop"}
	if err := cmd.Flags().Parse([]string{"listener-b:pipe-d"}); err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if err := listenercmd.StopPipelineCmd(cmd, h.Console); err != nil {
		t.Fatalf("StopPipelineCmd failed: %v", err)
	}

	calls := h.Recorder.Calls()
	if len(calls) != 1 || calls[0].Method != "StopPipeline" {
		t.Fatalf("calls = %#v, want only StopPipeline", calls)
	}
	req, ok := calls[0].Request.(*clientpb.CtrlPipeline)
	if !ok {
		t.Fatalf("request type = %T, want *clientpb.CtrlPipeline", calls[0].Request)
	}
	if req.Name != "pipe-d" || req.ListenerId != "listener-b" {
		t.Fatalf("stop request = %#v, want scoped pipe-d/listener-b", req)
	}
}

func TestDeletePipelineCmdParsesScopedNameWithoutCache(t *testing.T) {
	h := testsupport.NewClientHarness(t)

	cmd := &cobra.Command{Use: "delete"}
	if err := cmd.Flags().Parse([]string{"listener-c:pipe-e"}); err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if err := listenercmd.DeletePipelineCmd(cmd, h.Console); err != nil {
		t.Fatalf("DeletePipelineCmd failed: %v", err)
	}

	calls := h.Recorder.Calls()
	if len(calls) != 1 || calls[0].Method != "DeletePipeline" {
		t.Fatalf("calls = %#v, want only DeletePipeline", calls)
	}
	req, ok := calls[0].Request.(*clientpb.CtrlPipeline)
	if !ok {
		t.Fatalf("request type = %T, want *clientpb.CtrlPipeline", calls[0].Request)
	}
	if req.Name != "pipe-e" || req.ListenerId != "listener-c" {
		t.Fatalf("delete request = %#v, want scoped pipe-e/listener-c", req)
	}
}

func TestRestartPipelineCmdStopsThenStartsScopedPipeline(t *testing.T) {
	h := testsupport.NewClientHarness(t)

	cmd := &cobra.Command{Use: "restart"}
	if err := cmd.Flags().Parse([]string{"listener-a:pipe-a"}); err != nil {
		t.Fatalf("Parse failed: %v", err)
	}
	if err := listenercmd.RestartPipelineCmd(cmd, h.Console); err != nil {
		t.Fatalf("RestartPipelineCmd failed: %v", err)
	}

	calls := h.Recorder.Calls()
	if len(calls) != 2 || calls[0].Method != "StopPipeline" || calls[1].Method != "StartPipeline" {
		t.Fatalf("calls = %#v, want StopPipeline then StartPipeline", calls)
	}
	for _, call := range calls {
		req := call.Request.(*clientpb.CtrlPipeline)
		if req.Name != "pipe-a" || req.ListenerId != "listener-a" {
			t.Fatalf("%s request = %#v", call.Method, req)
		}
	}
}

func TestUpdatePipelineCmdSyncsCachedPipeline(t *testing.T) {
	h := testsupport.NewClientHarness(t)
	h.Console.Pipelines["listener-a:pipe-a"] = &clientpb.Pipeline{
		Name:       "pipe-a",
		ListenerId: "listener-a",
		Type:       consts.HTTPPipeline,
		Enable:     false,
		Parser:     "raw",
	}

	cmd := &cobra.Command{Use: "update"}
	cmd.Flags().Bool("enable", false, "")
	cmd.Flags().Bool("disable", false, "")
	cmd.Flags().String("cert-name", "", "")
	cmd.Flags().String("parser", "", "")
	if err := cmd.Flags().Parse([]string{"listener-a:pipe-a"}); err != nil {
		t.Fatalf("Parse failed: %v", err)
	}
	if err := cmd.Flags().Set("enable", "true"); err != nil {
		t.Fatalf("Set enable failed: %v", err)
	}
	if err := cmd.Flags().Set("cert-name", "web-cert"); err != nil {
		t.Fatalf("Set cert-name failed: %v", err)
	}
	if err := listenercmd.UpdatePipelineCmd(cmd, h.Console); err != nil {
		t.Fatalf("UpdatePipelineCmd failed: %v", err)
	}

	req, _ := testsupport.MustSingleCall[*clientpb.Pipeline](t, h, "SyncPipeline")
	if !req.Enable || req.CertName != "web-cert" || req.Name != "pipe-a" || req.ListenerId != "listener-a" {
		t.Fatalf("sync pipeline = %#v", req)
	}
}

func TestKillJobCmdStopsRunningPipeline(t *testing.T) {
	h := testsupport.NewClientHarness(t)
	h.Recorder.OnPipelines("ListJobs", func(context.Context, any) (*clientpb.Pipelines, error) {
		return &clientpb.Pipelines{Pipelines: []*clientpb.Pipeline{{Name: "job-a", ListenerId: "listener-a"}}}, nil
	})

	cmd := &cobra.Command{Use: "kill"}
	if err := cmd.Flags().Parse([]string{"listener-a:job-a"}); err != nil {
		t.Fatalf("Parse failed: %v", err)
	}
	if err := listenercmd.KillJobCmd(cmd, h.Console); err != nil {
		t.Fatalf("KillJobCmd failed: %v", err)
	}

	calls := h.Recorder.Calls()
	if len(calls) != 2 || calls[0].Method != "ListJobs" || calls[1].Method != "StopPipeline" {
		t.Fatalf("calls = %#v", calls)
	}
	req := calls[1].Request.(*clientpb.CtrlPipeline)
	if req.Name != "job-a" || req.ListenerId != "listener-a" {
		t.Fatalf("stop request = %#v", req)
	}
}
