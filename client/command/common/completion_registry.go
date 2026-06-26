package common

import (
	"sync"

	"github.com/carapace-sh/carapace"
	"github.com/chainreactors/malice-network/client/core"
	"github.com/spf13/cobra"
)

func init() {
	core.CompletionLookup = func(cmd *cobra.Command, flag string, argIndex int) (carapace.Action, bool) {
		if flag != "" {
			return GetFlagCompletion(cmd, flag)
		}
		return GetArgCompletion(cmd, argIndex)
	}
}

var (
	completionRegistryMu sync.RWMutex
	flagCompletions      = map[*cobra.Command]carapace.ActionMap{}
	argCompletions       = map[*cobra.Command]argCompletionSet{}
)

type argCompletionSet struct {
	anyAction carapace.Action
	hasAny    bool
	actions   []carapace.Action
}

func RegisterFlagCompletions(cmd *cobra.Command, comps carapace.ActionMap) {
	completionRegistryMu.Lock()
	defer completionRegistryMu.Unlock()
	existing, ok := flagCompletions[cmd]
	if !ok {
		existing = make(carapace.ActionMap)
		flagCompletions[cmd] = existing
	}
	for k, v := range comps {
		existing[k] = v
	}
}

func RegisterArgCompletions(cmd *cobra.Command, anyAction *carapace.Action, actions []carapace.Action) {
	completionRegistryMu.Lock()
	defer completionRegistryMu.Unlock()
	set := argCompletionSet{actions: actions}
	if anyAction != nil {
		set.anyAction = *anyAction
		set.hasAny = true
	}
	argCompletions[cmd] = set
}

func GetFlagCompletion(cmd *cobra.Command, flag string) (carapace.Action, bool) {
	completionRegistryMu.RLock()
	defer completionRegistryMu.RUnlock()
	comps, ok := flagCompletions[cmd]
	if !ok {
		return carapace.Action{}, false
	}
	action, ok := comps[flag]
	return action, ok
}

func GetArgCompletion(cmd *cobra.Command, index int) (carapace.Action, bool) {
	completionRegistryMu.RLock()
	defer completionRegistryMu.RUnlock()
	set, ok := argCompletions[cmd]
	if !ok || index < 0 {
		return carapace.Action{}, false
	}
	if index < len(set.actions) {
		return set.actions[index], true
	}
	if set.hasAny {
		return set.anyAction, true
	}
	return carapace.Action{}, false
}

func HasFlagCompletions(cmd *cobra.Command) bool {
	completionRegistryMu.RLock()
	defer completionRegistryMu.RUnlock()
	comps, ok := flagCompletions[cmd]
	return ok && len(comps) > 0
}

func ListFlagCompletionNames(cmd *cobra.Command) []string {
	completionRegistryMu.RLock()
	defer completionRegistryMu.RUnlock()
	comps, ok := flagCompletions[cmd]
	if !ok {
		return nil
	}
	names := make([]string, 0, len(comps))
	for k := range comps {
		names = append(names, k)
	}
	return names
}
