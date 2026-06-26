package website_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/chainreactors/IoM-go/consts"
	"github.com/chainreactors/IoM-go/proto/client/clientpb"
	"github.com/chainreactors/malice-network/client/command/testsupport"
)

func TestWebsiteInspectListsContent(t *testing.T) {
	h := testsupport.NewClientHarness(t)
	h.Console.Pipelines["listener-a:site-a"] = websitePipeline("site-a", "listener-a", 8080)
	h.Recorder.OnWebContents("ListWebContent", func(_ context.Context, req any) (*clientpb.WebContents, error) {
		website := req.(*clientpb.Website)
		if website.Name != "site-a" || website.ListenerId != "listener-a" {
			t.Fatalf("list content request = %#v", website)
		}
		return &clientpb.WebContents{Contents: []*clientpb.WebContent{{Id: "content-a", Path: "/index.html"}}}, nil
	})

	if err := h.ExecuteClient(consts.CommandWebsite, "inspect", "listener-a:site-a"); err != nil {
		t.Fatalf("website inspect failed: %v", err)
	}
	testsupport.MustSingleCall[*clientpb.Website](t, h, "ListWebContent")
}

func TestWebsiteRouteAddUsesContentRPC(t *testing.T) {
	h := testsupport.NewClientHarness(t)
	path := filepath.Join(t.TempDir(), "index.html")
	if err := os.WriteFile(path, []byte("hello"), 0600); err != nil {
		t.Fatalf("write content: %v", err)
	}

	if err := h.ExecuteClient(consts.CommandWebsite, "route", "add", path, "--website", "site-a", "--path", "/index.html"); err != nil {
		t.Fatalf("route add failed: %v", err)
	}

	req, _ := testsupport.MustSingleCall[*clientpb.Website](t, h, "AddWebsiteContent")
	if req.Name != "site-a" || string(req.Contents["/index.html"].Content) != "hello" {
		t.Fatalf("add content request = %#v", req)
	}
}

func TestWebsiteCertUsesTLSUpdate(t *testing.T) {
	h := testsupport.NewClientHarness(t)

	if err := h.ExecuteClient(consts.CommandWebsite, "cert", "site-a", "--listener", "listener-a", "--cert-name", "cert-a"); err != nil {
		t.Fatalf("website cert failed: %v", err)
	}

	req, _ := testsupport.MustSingleCall[*clientpb.PipelineTLSUpdate](t, h, "UpdateWebsiteTLS")
	if req.Name != "site-a" || req.ListenerId != "listener-a" || req.CertName != "cert-a" {
		t.Fatalf("tls update = %#v", req)
	}
}

func websitePipeline(name, listenerID string, port uint32) *clientpb.Pipeline {
	return &clientpb.Pipeline{
		Name:       name,
		ListenerId: listenerID,
		Enable:     true,
		CertName:   "cert-a",
		Body: &clientpb.Pipeline_Web{Web: &clientpb.Website{
			Name: name,
			Root: "/web",
			Port: port,
		}},
	}
}
