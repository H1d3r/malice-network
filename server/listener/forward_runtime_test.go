package listener

import (
	"context"
	"errors"
	"io"
	"net"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/chainreactors/IoM-go/consts"
	"github.com/chainreactors/IoM-go/proto/client/clientpb"
	"github.com/chainreactors/malice-network/helper/certs"
	"github.com/chainreactors/malice-network/server/forwardrpc"
	"github.com/chainreactors/malice-network/server/internal/certutils"
	"github.com/chainreactors/malice-network/server/internal/configs"
	"github.com/chainreactors/malice-network/server/internal/core"
	"google.golang.org/grpc"
	"gopkg.in/yaml.v3"
)

func reserveForwardPort(t testing.TB) uint16 {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("reserve port: %v", err)
	}
	defer ln.Close()
	return uint16(ln.Addr().(*net.TCPAddr).Port)
}

func writeForwardAuthConfig(t testing.TB) string {
	t.Helper()
	configs.UseTestPaths(t, filepath.Join(t.TempDir(), "malice"))
	if err := certutils.GenerateRootCert(); err != nil {
		t.Fatalf("GenerateRootCert failed: %v", err)
	}
	auth, _, err := certutils.GenerateListenerCert("127.0.0.1", "forward-listener-test", 0)
	if err != nil {
		t.Fatalf("GenerateListenerCert failed: %v", err)
	}
	data, err := yaml.Marshal(auth)
	if err != nil {
		t.Fatalf("marshal auth failed: %v", err)
	}
	path := filepath.Join(t.TempDir(), "listener.auth")
	if err := os.WriteFile(path, data, 0600); err != nil {
		t.Fatalf("write auth failed: %v", err)
	}
	return path
}

func writeClientOnlyForwardAuthConfig(t testing.TB) string {
	t.Helper()
	configs.UseTestPaths(t, filepath.Join(t.TempDir(), "malice"))
	if err := certutils.GenerateRootCert(); err != nil {
		t.Fatalf("GenerateRootCert failed: %v", err)
	}
	ca, caKey, err := certutils.GetCertificateAuthority()
	if err != nil {
		t.Fatalf("GetCertificateAuthority failed: %v", err)
	}
	auth, _, err := certutils.GenerateListenerCert("127.0.0.1", "forward-listener-test", 0)
	if err != nil {
		t.Fatalf("GenerateListenerCert failed: %v", err)
	}
	certPEM, keyPEM, err := certs.GenerateChildCert("127.0.0.1", true, ca, caKey)
	if err != nil {
		t.Fatalf("GenerateChildCert failed: %v", err)
	}
	auth.Certificate = string(certPEM)
	auth.PrivateKey = string(keyPEM)
	data, err := yaml.Marshal(auth)
	if err != nil {
		t.Fatalf("marshal auth failed: %v", err)
	}
	path := filepath.Join(t.TempDir(), "listener-client-only.auth")
	if err := os.WriteFile(path, data, 0600); err != nil {
		t.Fatalf("write auth failed: %v", err)
	}
	return path
}

func TestForwardListenerControlStreamStartsCustomPipeline(t *testing.T) {
	port := reserveForwardPort(t)
	cfg := &configs.ListenerConfig{
		Enable:    true,
		Name:      "forward-listener-test",
		Auth:      writeForwardAuthConfig(t),
		IP:        "127.0.0.1",
		Transport: configs.ListenerTransportForward,
		Forward: &configs.ForwardListenerConfig{
			ListenHost:  "127.0.0.1",
			ListenPort:  port,
			ConnectHost: "127.0.0.1",
			ConnectPort: port,
		},
	}
	runtime, err := NewForwardListener(cfg)
	if err != nil {
		t.Fatalf("NewForwardListener failed: %v", err)
	}
	t.Cleanup(func() { _ = runtime.Close() })

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	dialOptions, err := forwardrpc.DialOptions(cfg.ForwardConfigOrDefault().ConnectHost)
	if err != nil {
		t.Fatalf("build forward dial options: %v", err)
	}
	conn, err := grpc.DialContext(ctx, cfg.ForwardConfigOrDefault().ConnectAddress(),
		dialOptions...,
	)
	if err != nil {
		t.Fatalf("dial forward listener: %v", err)
	}
	t.Cleanup(func() { _ = conn.Close() })

	stream, err := forwardrpc.NewForwardListenerClient(conn).ControlStream(ctx)
	if err != nil {
		t.Fatalf("open control stream: %v", err)
	}

	pipeline := &clientpb.Pipeline{
		Name:       "custom-forward",
		ListenerId: cfg.Name,
		Enable:     true,
		Type:       consts.CustomPipeline,
		Body: &clientpb.Pipeline_Custom{
			Custom: &clientpb.CustomPipeline{Name: "custom-forward", ListenerId: cfg.Name},
		},
		Tls:        &clientpb.TLS{},
		Encryption: []*clientpb.Encryption{},
		Secure:     &clientpb.Secure{},
	}
	if err := stream.Send(&clientpb.JobCtrl{
		Id:   1,
		Ctrl: consts.CtrlPipelineStart,
		Job:  &clientpb.Job{Name: pipeline.Name, Pipeline: pipeline},
	}); err != nil {
		t.Fatalf("send start ctrl: %v", err)
	}
	status, err := stream.Recv()
	if err != nil {
		t.Fatalf("recv status: %v", err)
	}
	if status.Status != consts.CtrlStatusSuccess {
		t.Fatalf("status = %#v, want success", status)
	}
	if got := runtime.lns.pipelines.Get(pipeline.Name); got == nil {
		t.Fatalf("pipeline %s was not started", pipeline.Name)
	}
}

func TestForwardStreamRegistryKeepsStreamUntilPipelineStop(t *testing.T) {
	listenerID := "forward-registry-listener"
	pipelineID := "forward-registry-pipeline"
	registry := newForwardStreamRegistry()
	lns := &listener{
		Name:        listenerID,
		IP:          "127.0.0.1",
		pipelines:   core.NewPipelines(),
		websites:    map[string]*Website{},
		pipelineRPC: &forwardPipelineRPC{listenerID: listenerID, registry: registry},
	}

	pipeline := &clientpb.Pipeline{
		Name:       pipelineID,
		ListenerId: listenerID,
		Enable:     true,
		Type:       consts.TCPPipeline,
		Body: &clientpb.Pipeline_Tcp{
			Tcp: &clientpb.TCPPipeline{
				Name:       pipelineID,
				ListenerId: listenerID,
				Host:       "127.0.0.1",
				Port:       0,
			},
		},
		Tls:        &clientpb.TLS{},
		Encryption: []*clientpb.Encryption{},
		Secure:     &clientpb.Secure{},
	}
	started, err := lns.startPipeline(pipeline)
	if err != nil {
		t.Fatalf("startPipeline failed: %v", err)
	}
	t.Cleanup(func() { _ = started.Close() })

	beforeDisconnect := registry.get(listenerID, pipelineID)
	taskStream := &mockTaskStream{
		ctx:     context.Background(),
		recvErr: io.EOF,
	}
	if err := beforeDisconnect.serve(taskStream); err != nil {
		t.Fatalf("serve EOF error = %v", err)
	}
	if afterDisconnect := registry.get(listenerID, pipelineID); afterDisconnect != beforeDisconnect {
		t.Fatal("TaskStream disconnect should keep the existing registry stream")
	}

	status := lns.handleJobCtrl(&clientpb.JobCtrl{
		Id:   1,
		Ctrl: consts.CtrlPipelineStop,
		Job:  &clientpb.Job{Name: pipelineID, Pipeline: pipeline},
	})
	if status == nil || status.Status != consts.CtrlStatusSuccess {
		t.Fatalf("stop status = %#v, want success", status)
	}
	afterStop := registry.get(listenerID, pipelineID)
	if afterStop == beforeDisconnect {
		t.Fatal("pipeline stop should remove the old registry stream")
	}
	select {
	case err := <-recvForwardStream(beforeDisconnect):
		if !errors.Is(err, io.EOF) {
			t.Fatalf("old stream Recv error = %v, want EOF", err)
		}
	case <-time.After(time.Second):
		t.Fatal("old stream Recv did not unblock after pipeline stop")
	}
}

func TestForwardStreamRegistryCleanupOnListenerClose(t *testing.T) {
	listenerID := "forward-close-listener"
	pipelineID := "forward-close-pipeline"
	registry := newForwardStreamRegistry()
	lns := &listener{
		Name:        listenerID,
		IP:          "127.0.0.1",
		pipelines:   core.NewPipelines(),
		websites:    map[string]*Website{},
		pipelineRPC: &forwardPipelineRPC{listenerID: listenerID, registry: registry},
	}

	pipeline := &clientpb.Pipeline{
		Name:       pipelineID,
		ListenerId: listenerID,
		Enable:     true,
		Type:       consts.CustomPipeline,
		Body: &clientpb.Pipeline_Custom{
			Custom: &clientpb.CustomPipeline{Name: pipelineID, ListenerId: listenerID},
		},
		Tls:        &clientpb.TLS{},
		Encryption: []*clientpb.Encryption{},
		Secure:     &clientpb.Secure{},
	}
	lns.pipelines.Add(NewCustomPipeline(pipeline))
	beforeClose := registry.get(listenerID, pipelineID)

	if err := lns.Close(); err != nil {
		t.Fatalf("listener Close failed: %v", err)
	}
	afterClose := registry.get(listenerID, pipelineID)
	if afterClose == beforeClose {
		t.Fatal("listener Close should remove the old registry stream")
	}
}

func recvForwardStream(stream *forwardLocalStream) <-chan error {
	errCh := make(chan error, 1)
	go func() {
		_, err := stream.Recv()
		errCh <- err
	}()
	return errCh
}

func TestForwardListenerRejectsClientOnlyAuth(t *testing.T) {
	port := reserveForwardPort(t)
	cfg := &configs.ListenerConfig{
		Enable:    true,
		Name:      "forward-listener-test",
		Auth:      writeClientOnlyForwardAuthConfig(t),
		IP:        "127.0.0.1",
		Transport: configs.ListenerTransportForward,
		Forward: &configs.ForwardListenerConfig{
			ListenHost: "127.0.0.1",
			ListenPort: port,
		},
	}
	_, err := NewForwardListener(cfg)
	if err == nil {
		t.Fatal("NewForwardListener succeeded with client-only auth, want serverAuth error")
	}
	if !strings.Contains(err.Error(), "serverAuth") {
		t.Fatalf("NewForwardListener error = %q, want serverAuth", err.Error())
	}
}
