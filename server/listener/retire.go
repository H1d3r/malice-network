package listener

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/chainreactors/IoM-go/proto/client/clientpb"
	"github.com/chainreactors/logs"
	"github.com/chainreactors/malice-network/server/internal/configs"
)

const listenerRetireShutdownDelay = 100 * time.Millisecond

func (lns *listener) handleRetire(req *clientpb.ListenerRetire) error {
	if lns == nil {
		return fmt.Errorf("listener is nil")
	}
	if req == nil {
		req = &clientpb.ListenerRetire{}
	}
	if listenerID := strings.TrimSpace(req.GetListenerId()); listenerID != "" && listenerID != lns.ID() {
		return fmt.Errorf("retire target %s does not match listener %s", listenerID, lns.ID())
	}
	if req.GetPurgeConfig() {
		if err := removeRetireFile(configs.CurrentServerConfigFilename, "config"); err != nil {
			return err
		}
	}
	if req.GetPurgeAuth() {
		if lns.cfg == nil {
			return fmt.Errorf("listener config is nil")
		}
		if err := removeRetireFile(lns.cfg.Auth, "auth"); err != nil {
			return err
		}
	}
	lns.scheduleRetireShutdown()
	return nil
}

func (lns *listener) scheduleRetireShutdown() {
	lns.retireOnce.Do(func() {
		go func() {
			time.Sleep(listenerRetireShutdownDelay)
			shutdown := lns.Close
			if lns.shutdown != nil {
				shutdown = lns.shutdown
			}
			if err := shutdown(); err != nil {
				logs.Log.Errorf("listener.%s - retire_shutdown_failed error=%q", lns.ID(), err)
			}
		}()
	})
}

func removeRetireFile(path, kind string) error {
	path = strings.TrimSpace(path)
	if path == "" {
		return fmt.Errorf("%s path is empty", kind)
	}
	info, err := os.Stat(path)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("stat %s file %s: %w", kind, path, err)
	}
	if info.IsDir() {
		return fmt.Errorf("%s path %s is a directory", kind, path)
	}
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("remove %s file %s: %w", kind, path, err)
	}
	return nil
}
