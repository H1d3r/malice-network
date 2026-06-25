package build_test

import (
	"testing"

	"github.com/chainreactors/IoM-go/consts"
	"github.com/chainreactors/IoM-go/proto/client/clientpb"
	"github.com/chainreactors/malice-network/client/command/testsupport"
)

func TestArtifactCommentCommandForwardsUpdateArtifact(t *testing.T) {
	h := testsupport.NewClientHarness(t)

	if err := h.ExecuteClient(consts.CommandArtifact, "comment", "demo-artifact", "new comment"); err != nil {
		t.Fatalf("artifact comment failed: %v", err)
	}

	req, _ := testsupport.MustSingleCall[*clientpb.Artifact](t, h, "UpdateArtifact")
	if req.Name != "demo-artifact" {
		t.Fatalf("artifact name = %q, want demo-artifact", req.Name)
	}
	if req.Comment != "new comment" {
		t.Fatalf("comment = %q, want new comment", req.Comment)
	}
	testsupport.RequireNoSessionEvents(t, h)
}
