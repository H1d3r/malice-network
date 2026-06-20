package rpc

import (
	"context"
	"fmt"
	"sync"

	"github.com/chainreactors/IoM-go/consts"
	"github.com/chainreactors/IoM-go/proto/client/clientpb"
	"github.com/chainreactors/IoM-go/proto/implant/implantpb"
	"github.com/chainreactors/IoM-go/types"
	"github.com/chainreactors/utils/pty"
)

var implantPTYRouters sync.Map

func (rpc *Server) getImplantPTY(session string) (*pty.Router, *ImplantPTYManager) {
	if v, ok := implantPTYRouters.Load(session); ok {
		entry := v.(*implantPTYEntry)
		return entry.router, entry.mgr
	}
	mgr := NewImplantPTYManager()
	router := pty.NewRouter(mgr, pty.WithOpener("shell", mgr.OpenShell(rpc)))
	entry := &implantPTYEntry{router: router, mgr: mgr}
	actual, _ := implantPTYRouters.LoadOrStore(session, entry)
	e := actual.(*implantPTYEntry)
	return e.router, e.mgr
}

type implantPTYEntry struct {
	router *pty.Router
	mgr    *ImplantPTYManager
}

func (rpc *Server) PtyRequest(ctx context.Context, req *implantpb.PtyRequest) (*clientpb.Task, error) {
	session, err := getSession(ctx)
	if err != nil {
		return nil, err
	}

	router, _ := rpc.getImplantPTY(session.ID)
	frame := protoToFrame(req)

	var result pty.Frame
	router.Handle(ctx, frame, func(out pty.Frame) {
		result = out
	})

	if result.Type == pty.FrameError {
		return nil, fmt.Errorf("pty: %s", result.Error)
	}

	if result.Type == pty.FrameOpened && result.Session != nil {
		greq, err := newGenericRequest(ctx, req)
		if err == nil {
			return greq.Task.ToProtobuf(), nil
		}
	}

	greq, err := newGenericRequest(ctx, req)
	if err != nil {
		return nil, err
	}

	switch req.GetType() {
	case consts.ModulePtyStop:
		return greq.Task.ToProtobuf(), nil
	default:
		ch, err := rpc.GenericHandler(ctx, greq)
		if err != nil {
			return nil, err
		}
		greq.HandlerResponse(ch, types.MsgPtyResponse)
		return greq.Task.ToProtobuf(), nil
	}
}

func protoToFrame(req *implantpb.PtyRequest) pty.Frame {
	switch req.GetType() {
	case consts.ModulePtyStart:
		return pty.Frame{
			Type:    pty.FrameOpen,
			Kind:    "shell",
			Name:    req.Shell,
			Command: req.Shell,
			Cols:    int(req.Cols),
			Rows:    int(req.Rows),
		}
	case consts.ModulePtyInput:
		return pty.Frame{
			Type:      pty.FrameInput,
			SessionID: req.SessionId,
			Data:      append(req.InputData, []byte(req.InputText)...),
		}
	case consts.ModulePtyStop:
		return pty.Frame{
			Type:      pty.FrameKill,
			SessionID: req.SessionId,
		}
	case "resize":
		return pty.Frame{
			Type:      pty.FrameResize,
			SessionID: req.SessionId,
			Cols:      int(req.Cols),
			Rows:      int(req.Rows),
		}
	default:
		return pty.Frame{
			Type:      pty.FrameInput,
			SessionID: req.SessionId,
			Data:      append(req.InputData, []byte(req.InputText)...),
		}
	}
}
