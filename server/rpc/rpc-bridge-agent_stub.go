//go:build !bridge_agent_proto
// +build !bridge_agent_proto

package rpc

import (
	"context"
	"errors"

	"github.com/chainreactors/IoM-go/proto/client/clientpb"
	"github.com/chainreactors/IoM-go/proto/implant/implantpb"
)

var errBridgeAgentRPCUnavailable = errors.New("bridge agent RPC is unavailable in this build")

func (rpc *Server) BridgeAgentChat(ctx context.Context, req *implantpb.BridgeAgentRequest) (*clientpb.Task, error) {
	return nil, errBridgeAgentRPCUnavailable
}
