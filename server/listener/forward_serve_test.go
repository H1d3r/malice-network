package listener

import (
	"context"
	"errors"
	"io"
	"runtime"
	"testing"
	"time"

	"github.com/chainreactors/IoM-go/proto/client/clientpb"
	"google.golang.org/grpc/metadata"
)

type mockTaskStream struct {
	ctx      context.Context
	recvErr  error
	sendErr  error
	recvReq  *clientpb.SpiteRequest
	recvOnce bool
}

func (m *mockTaskStream) Send(*clientpb.SpiteRequest) error {
	if m.sendErr != nil {
		return m.sendErr
	}
	return nil
}
func (m *mockTaskStream) Recv() (*clientpb.SpiteRequest, error) {
	if m.recvReq != nil && !m.recvOnce {
		m.recvOnce = true
		return m.recvReq, nil
	}
	if m.recvErr != nil {
		return nil, m.recvErr
	}
	<-m.ctx.Done()
	return nil, io.EOF
}
func (m *mockTaskStream) SetHeader(metadata.MD) error  { return nil }
func (m *mockTaskStream) SendHeader(metadata.MD) error { return nil }
func (m *mockTaskStream) SetTrailer(metadata.MD)       {}
func (m *mockTaskStream) Context() context.Context     { return m.ctx }
func (m *mockTaskStream) SendMsg(interface{}) error    { return nil }
func (m *mockTaskStream) RecvMsg(interface{}) error    { return nil }

func TestServeGoroutinesExitOnRecvError(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	stream := &forwardLocalStream{
		listenerID: "test-lns",
		pipelineID: "test-pipe",
		requests:   make(chan *clientpb.SpiteRequest, 255),
		events:     make(chan *clientpb.SpiteRequest, 255),
	}

	mock := &mockTaskStream{
		ctx:     ctx,
		recvErr: io.EOF,
	}

	before := runtime.NumGoroutine()
	err := stream.serve(mock)
	if err != nil {
		t.Fatalf("serve returned error for EOF: %v", err)
	}

	time.Sleep(50 * time.Millisecond)
	after := runtime.NumGoroutine()

	leaked := after - before
	if leaked > 1 {
		t.Fatalf("goroutine leak detected: %d goroutines before serve, %d after (+%d)", before, after, leaked)
	}
}

func TestServeCancelsBlockedRequestSendOnSendError(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	stream := &forwardLocalStream{
		listenerID: "test-lns",
		pipelineID: "test-pipe",
		requests:   make(chan *clientpb.SpiteRequest),
		events:     make(chan *clientpb.SpiteRequest, 1),
	}
	stream.events <- &clientpb.SpiteRequest{ListenerId: "test-lns"}

	sendErr := errors.New("send failed")
	mock := &mockTaskStream{
		ctx:     ctx,
		sendErr: sendErr,
		recvReq: &clientpb.SpiteRequest{ListenerId: "test-lns"},
	}

	err := stream.serve(mock)
	if !errors.Is(err, sendErr) {
		t.Fatalf("serve error = %v, want %v", err, sendErr)
	}

	time.Sleep(20 * time.Millisecond)
	select {
	case req := <-stream.requests:
		t.Fatalf("request sender remained blocked after serve returned and delivered %#v", req)
	default:
	}
}
