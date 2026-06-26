package website_test

import (
	"context"
	"os"
	"path/filepath"
	"strings"
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

func TestWebsiteExportWritesMetadata(t *testing.T) {
	h := testsupport.NewClientHarness(t)
	h.Console.Pipelines["listener-a:site-a"] = websitePipeline("site-a", "listener-a", 8080)
	h.Recorder.OnWebContents("ListWebContent", func(context.Context, any) (*clientpb.WebContents, error) {
		return &clientpb.WebContents{Contents: []*clientpb.WebContent{{Id: "content-a", Path: "/index.html", ContentType: "text/html"}}}, nil
	})
	output := filepath.Join(t.TempDir(), "site.json")

	if err := h.ExecuteClient(consts.CommandWebsite, "export", "listener-a:site-a", "-o", output); err != nil {
		t.Fatalf("website export failed: %v", err)
	}
	data, err := os.ReadFile(output)
	if err != nil {
		t.Fatalf("read export: %v", err)
	}
	if !containsAll(string(data), `"name": "site-a"`, `"listenerId": "listener-a"`, `"/index.html"`) {
		t.Fatalf("export data = %s", data)
	}
}

func TestWebsiteImportRegistersAndStartsWebsite(t *testing.T) {
	h := testsupport.NewClientHarness(t)
	path := filepath.Join(t.TempDir(), "site.json")
	if err := os.WriteFile(path, []byte(`{"name":"site-a","listenerId":"listener-a","port":8080,"root":"/web"}`), 0600); err != nil {
		t.Fatalf("write export: %v", err)
	}

	if err := h.ExecuteClient(consts.CommandWebsite, "import", path); err != nil {
		t.Fatalf("website import failed: %v", err)
	}

	calls := h.Recorder.Calls()
	if len(calls) != 2 || calls[0].Method != "RegisterWebsite" || calls[1].Method != "StartWebsite" {
		t.Fatalf("calls = %#v", calls)
	}
	req := calls[0].Request.(*clientpb.Pipeline)
	if req.Name != "site-a" || req.ListenerId != "listener-a" || req.GetWeb().Port != 8080 {
		t.Fatalf("register request = %#v", req)
	}
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

func containsAll(s string, parts ...string) bool {
	for _, part := range parts {
		if !strings.Contains(s, part) {
			return false
		}
	}
	return true
}
