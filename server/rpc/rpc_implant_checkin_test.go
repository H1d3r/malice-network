package rpc

import (
	"context"
	"testing"

	implantpb "github.com/chainreactors/IoM-go/proto/implant/implantpb"
	"google.golang.org/grpc/metadata"
)

func TestCheckinReturnsErrorForNonexistentSession(t *testing.T) {
	_ = newRPCTestEnv(t)

	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs(
		"session_id", "nonexistent-session-id",
	))

	_, err := (&Server{}).Checkin(ctx, &implantpb.Ping{})
	if err == nil {
		t.Fatal("Checkin should return error for nonexistent session, got nil")
	}
}
