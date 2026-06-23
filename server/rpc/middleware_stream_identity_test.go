package rpc

import (
	"context"
	"testing"

	"github.com/chainreactors/IoM-go/mtls"
	"github.com/chainreactors/IoM-go/proto/client/rootpb"
	"github.com/chainreactors/malice-network/server/internal/db"
	"github.com/chainreactors/malice-network/server/internal/db/models"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestIdentityServerStreamRejectsRemovedOperatorOnNextMessage(t *testing.T) {
	initForwardRPCTestDB(t)
	if err := db.SeedDefaultAuthzRules(); err != nil {
		t.Fatalf("SeedDefaultAuthzRules failed: %v", err)
	}
	ruleCache.Invalidate()

	const (
		operatorName = "stream-removed-client"
		fingerprint  = "stream-removed-fingerprint"
	)
	if err := db.CreateOperator(&models.Operator{
		Name:        operatorName,
		Type:        mtls.Client,
		Role:        models.RoleOperator,
		Fingerprint: fingerprint,
	}); err != nil {
		t.Fatalf("CreateOperator failed: %v", err)
	}
	opCache.Invalidate()

	wrapped := &identityServerStream{
		ServerStream: &testRPCServerStream{},
		identity:     &PeerIdentity{Fingerprint: fingerprint},
	}
	if err := wrapped.RecvMsg(&rootpb.Response{}); err != nil {
		t.Fatalf("RecvMsg before RemoveClient returned error: %v", err)
	}

	if _, err := (&Server{}).RemoveClient(context.Background(), &rootpb.Operator{Args: []string{operatorName}}); err != nil {
		t.Fatalf("RemoveClient failed: %v", err)
	}

	err := wrapped.RecvMsg(&rootpb.Response{})
	if status.Code(err) != codes.Unauthenticated {
		t.Fatalf("RecvMsg after RemoveClient error = %v, want Unauthenticated", err)
	}
}

func TestIdentityServerStreamRejectsRevokedOperatorOnNextMessage(t *testing.T) {
	initForwardRPCTestDB(t)
	const (
		operatorName = "stream-revoked-client"
		fingerprint  = "stream-revoked-fingerprint"
	)
	if err := db.CreateOperator(&models.Operator{
		Name:        operatorName,
		Type:        mtls.Client,
		Role:        models.RoleOperator,
		Fingerprint: fingerprint,
	}); err != nil {
		t.Fatalf("CreateOperator failed: %v", err)
	}
	opCache.Invalidate()

	wrapped := &identityServerStream{
		ServerStream: &testRPCServerStream{},
		identity:     &PeerIdentity{Fingerprint: fingerprint},
	}
	if err := wrapped.SendMsg(&rootpb.Response{}); err != nil {
		t.Fatalf("SendMsg before revoke returned error: %v", err)
	}

	if err := db.RevokeOperator(operatorName); err != nil {
		t.Fatalf("RevokeOperator failed: %v", err)
	}
	opCache.InvalidateByName(operatorName)

	err := wrapped.SendMsg(&rootpb.Response{})
	if status.Code(err) != codes.Unauthenticated {
		t.Fatalf("SendMsg after revoke error = %v, want Unauthenticated", err)
	}
}
