//go:build !bridge_agent_proto
// +build !bridge_agent_proto

package agent

import (
	"errors"

	"github.com/chainreactors/IoM-go/client"
	"github.com/chainreactors/IoM-go/proto/client/clientpb"
	"github.com/chainreactors/IoM-go/proto/services/clientrpc"
)

var errBridgeAgentUnavailable = errors.New("bridge agent is unavailable in this build: current proto definitions do not include the required RPC/messages")

func BridgeAgentAvailable() bool {
	return false
}

func BridgeAgentChat(rpc clientrpc.MaliceRPCClient, sess *client.Session,
	text, model, provider string, maxTurns uint32) (*clientpb.Task, error) {
	return nil, errBridgeAgentUnavailable
}
