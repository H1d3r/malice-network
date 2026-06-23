package rpc

import (
	"context"
	"testing"

	"github.com/chainreactors/IoM-go/proto/client/rootpb"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestAddClientRejectsEmptyArgs(t *testing.T) {
	for _, args := range [][]string{nil, {}} {
		_, err := (&Server{}).AddClient(context.Background(), &rootpb.Operator{Args: args})
		if status.Code(err) != codes.InvalidArgument {
			t.Fatalf("AddClient(args=%v) error = %v, want InvalidArgument", args, err)
		}
	}
}

func TestRemoveClientRejectsEmptyArgs(t *testing.T) {
	for _, args := range [][]string{nil, {}} {
		_, err := (&Server{}).RemoveClient(context.Background(), &rootpb.Operator{Args: args})
		if status.Code(err) != codes.InvalidArgument {
			t.Fatalf("RemoveClient(args=%v) error = %v, want InvalidArgument", args, err)
		}
	}
}

func TestAddListenerRejectsEmptyArgs(t *testing.T) {
	for _, args := range [][]string{nil, {}} {
		_, err := (&Server{}).AddListener(context.Background(), &rootpb.Operator{Args: args})
		if status.Code(err) != codes.InvalidArgument {
			t.Fatalf("AddListener(args=%v) error = %v, want InvalidArgument", args, err)
		}
	}
}
