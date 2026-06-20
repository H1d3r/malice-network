package rpc

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/chainreactors/IoM-go/consts"
	"github.com/chainreactors/IoM-go/proto/implant/implantpb"
	"github.com/chainreactors/IoM-go/types"
	"github.com/chainreactors/logs"
	"github.com/chainreactors/malice-network/server/internal/core"
	"github.com/chainreactors/utils/pty"
)

type implantPTYSession struct {
	info   pty.Info
	writer *core.SpiteStreamWriter
	greq   *GenericRequest
	output *pty.OutputBuffer
	done   chan struct{}
	once   sync.Once
}

type ImplantPTYManager struct {
	mu       sync.Mutex
	sessions map[string]*implantPTYSession
}

func NewImplantPTYManager() *ImplantPTYManager {
	return &ImplantPTYManager{sessions: make(map[string]*implantPTYSession)}
}

func (m *ImplantPTYManager) List() []pty.Info {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := make([]pty.Info, 0, len(m.sessions))
	for _, s := range m.sessions {
		out = append(out, s.info)
	}
	return out
}

func (m *ImplantPTYManager) Get(id string) (pty.Info, bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	s, ok := m.sessions[id]
	if !ok {
		return pty.Info{}, false
	}
	return s.info, true
}

func (m *ImplantPTYManager) Write(id string, data []byte) error {
	s := m.get(id)
	if s == nil {
		return fmt.Errorf("no such pty session: %s", id)
	}
	req := &implantpb.PtyRequest{
		Type:      consts.ModulePtyInput,
		SessionId: id,
		InputData: data,
	}
	return m.sendToImplant(s, req)
}

func (m *ImplantPTYManager) Resize(id string, cols, rows int) error {
	s := m.get(id)
	if s == nil {
		return fmt.Errorf("no such pty session: %s", id)
	}
	req := &implantpb.PtyRequest{
		Type:      "resize",
		SessionId: id,
		Cols:      uint32(cols),
		Rows:      uint32(rows),
	}
	return m.sendToImplant(s, req)
}

func (m *ImplantPTYManager) Kill(id string) error {
	s := m.get(id)
	if s == nil {
		return fmt.Errorf("no such pty session: %s", id)
	}
	req := &implantpb.PtyRequest{
		Type:      consts.ModulePtyStop,
		SessionId: id,
	}
	err := m.sendToImplant(s, req)
	m.finish(id, pty.StateKilled, "user request")
	return err
}

func (m *ImplantPTYManager) SnapshotBytes(id string, n int) ([]byte, int64, error) {
	s := m.get(id)
	if s == nil {
		return nil, 0, fmt.Errorf("no such pty session: %s", id)
	}
	data, offset := s.output.TailRawBytesWithOffset(n)
	return data, offset, nil
}

func (m *ImplantPTYManager) MonitorFrom(ctx context.Context, id string, offset int64, interval time.Duration, push func([]byte)) error {
	s := m.get(id)
	if s == nil {
		return fmt.Errorf("no such pty session: %s", id)
	}
	if interval <= 0 {
		interval = 50 * time.Millisecond
	}
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-s.done:
				if data, _, _ := s.output.ReadSinceLimit(offset, 0); len(data) > 0 {
					push(data)
				}
				return
			case <-ticker.C:
				data, newOff, _, err := s.output.ReadSinceLimit(offset, 0)
				if err != nil {
					return
				}
				offset = newOff
				if len(data) > 0 {
					push(data)
				}
			}
		}
	}()
	return nil
}

func (m *ImplantPTYManager) Wait(ctx context.Context, id string, timeout time.Duration) (pty.Info, error) {
	s := m.get(id)
	if s == nil {
		return pty.Info{}, fmt.Errorf("no such pty session: %s", id)
	}
	select {
	case <-ctx.Done():
		return s.info, ctx.Err()
	case <-s.done:
		m.mu.Lock()
		info := s.info
		m.mu.Unlock()
		return info, nil
	}
}

func (m *ImplantPTYManager) OpenShell(rpc *Server) pty.OpenFunc {
	return func(ctx context.Context, spec pty.OpenSpec) (pty.OpenResult, error) {
		greq, err := newGenericRequest(ctx, &implantpb.PtyRequest{
			Type:  consts.ModulePtyStart,
			Shell: spec.Command,
			Cols:  uint32(spec.Cols),
			Rows:  uint32(spec.Rows),
			Params: map[string]string{
				"streaming": "true",
			},
		})
		if err != nil {
			return pty.OpenResult{}, err
		}
		greq.Count = -1

		writer, respCh, err := rpc.StreamGenericHandler(ctx, greq)
		if err != nil {
			return pty.OpenResult{}, err
		}

		id := fmt.Sprintf("pty-%d", time.Now().UnixNano())
		info := pty.Info{
			ID:        id,
			Kind:      "shell",
			Name:      spec.Name,
			Command:   spec.Command,
			State:     pty.StateRunning,
			StartedAt: time.Now(),
		}

		s := &implantPTYSession{
			info:   info,
			writer: writer,
			greq:   greq,
			output: pty.NewOutputBuffer(pty.DefaultBufferCap),
			done:   make(chan struct{}),
		}

		m.mu.Lock()
		m.sessions[id] = s
		m.mu.Unlock()

		go m.pumpImplantOutput(id, s, respCh)

		return pty.OpenResult{Info: info}, nil
	}
}

func (m *ImplantPTYManager) pumpImplantOutput(id string, s *implantPTYSession, respCh <-chan *implantpb.Spite) {
	defer func() {
		s.writer.Close()
		m.finish(id, pty.StateCompleted, "")
	}()

	for {
		resp, ok := recvSpite(s.greq.Task.Ctx, respCh)
		if !ok || resp == nil {
			return
		}

		_ = s.greq.HandlerSpite(resp)
		s.greq.Task.Finish(resp, "")

		ptyResp := resp.GetPtyResponse()
		if ptyResp == nil {
			moduleResp := resp.GetResponse()
			if moduleResp != nil && moduleResp.GetError() != "" &&
				strings.Contains(moduleResp.GetError(), "session") &&
				strings.Contains(moduleResp.GetError(), "closed") {
				return
			}
			continue
		}

		if len(ptyResp.OutputData) > 0 {
			s.output.Write(ptyResp.OutputData)
		}
		if ptyResp.OutputText != "" {
			s.output.Write([]byte(ptyResp.OutputText))
		}

		if !ptyResp.SessionActive {
			s.greq.Task.Finish(resp, "Shell session ended")
			return
		}
	}
}

func (m *ImplantPTYManager) get(id string) *implantPTYSession {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.sessions[id]
}

func (m *ImplantPTYManager) finish(id string, state pty.State, cause string) {
	m.mu.Lock()
	s, ok := m.sessions[id]
	if !ok {
		m.mu.Unlock()
		return
	}
	s.info.State = state
	s.info.KillCause = cause
	s.info.EndedAt = time.Now()
	m.mu.Unlock()

	s.once.Do(func() {
		close(s.done)
		logs.Log.Debugf("[pty] session %s finished: %s", id, state)
	})
}

func (m *ImplantPTYManager) sendToImplant(s *implantPTYSession, req *implantpb.PtyRequest) error {
	spite := &implantpb.Spite{
		Name:   types.MsgPty.String(),
		Body:   &implantpb.Spite_PtyRequest{PtyRequest: req},
		TaskId: s.greq.Task.Id,
	}
	return s.writer.Send(spite)
}

func (m *ImplantPTYManager) Close() {
	m.mu.Lock()
	ids := make([]string, 0, len(m.sessions))
	for id := range m.sessions {
		ids = append(ids, id)
	}
	m.mu.Unlock()
	for _, id := range ids {
		m.Kill(id)
	}
}
