package website

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"

	"github.com/chainreactors/IoM-go/proto/client/clientpb"
	"github.com/chainreactors/malice-network/client/core"
	"github.com/chainreactors/tui"
	"github.com/spf13/cobra"
)

type websiteExportFile struct {
	Name       string               `json:"name"`
	ListenerID string               `json:"listenerId"`
	Port       uint32               `json:"port"`
	Root       string               `json:"root"`
	Auth       string               `json:"auth,omitempty"`
	Enable     bool                 `json:"enable"`
	CertName   string               `json:"certName,omitempty"`
	Contents   []websiteContentItem `json:"contents,omitempty"`
}

type websiteContentItem struct {
	ID          string `json:"id,omitempty"`
	Path        string `json:"path"`
	Name        string `json:"name,omitempty"`
	ContentType string `json:"contentType,omitempty"`
	Size        uint64 `json:"size,omitempty"`
	Comment     string `json:"comment,omitempty"`
	Auth        string `json:"auth,omitempty"`
}

func InspectWebsiteCmd(cmd *cobra.Command, con *core.Console) error {
	name := cmd.Flags().Arg(0)
	website, err := findWebsite(con, name)
	if err != nil {
		return err
	}
	contents, err := listWebsiteContents(con, website)
	if err != nil {
		return err
	}
	web := website.GetWeb()
	tui.RenderKVWithOptions(map[string]interface{}{
		"Name":       website.GetName(),
		"ListenerID": website.GetListenerId(),
		"Port":       strconv.Itoa(int(web.GetPort())),
		"Root":       web.GetRoot(),
		"Enable":     website.GetEnable(),
		"CertName":   website.GetCertName(),
		"Contents":   len(contents.GetContents()),
	}, []string{"Name", "ListenerID", "Port", "Root", "Enable", "CertName", "Contents"}, tui.KVOptions{ShowHeader: true})
	return nil
}

func ExportWebsiteCmd(cmd *cobra.Command, con *core.Console) error {
	name := cmd.Flags().Arg(0)
	output, _ := cmd.Flags().GetString("output")
	website, err := findWebsite(con, name)
	if err != nil {
		return err
	}
	contents, err := listWebsiteContents(con, website)
	if err != nil {
		return err
	}
	exported := websiteToExport(website, contents)
	data, err := json.MarshalIndent(exported, "", "  ")
	if err != nil {
		return err
	}
	if output == "" || output == "-" {
		con.Log.Console(string(data) + "\n")
		return nil
	}
	if err := os.WriteFile(output, data, 0600); err != nil {
		return err
	}
	con.Log.Infof("website %s exported to %s\n", name, output)
	return nil
}

func ImportWebsiteCmd(cmd *cobra.Command, con *core.Console) error {
	path := cmd.Flags().Arg(0)
	nameOverride, _ := cmd.Flags().GetString("name")
	listenerOverride, _ := cmd.Flags().GetString("listener")
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	var exported websiteExportFile
	if err := json.Unmarshal(data, &exported); err != nil {
		return err
	}
	if nameOverride != "" {
		exported.Name = nameOverride
	}
	if listenerOverride != "" {
		exported.ListenerID = listenerOverride
	}
	if exported.Name == "" {
		return fmt.Errorf("website name is required")
	}
	if exported.Port == 0 {
		return fmt.Errorf("website port is required")
	}
	var tlsConfig *clientpb.TLS
	if exported.CertName != "" {
		tlsConfig = &clientpb.TLS{Enable: true}
	}
	if err := NewWebsite(con, exported.Name, exported.Root, "", exported.Port, exported.ListenerID, exported.CertName, tlsConfig, exported.Auth); err != nil {
		return err
	}
	if len(exported.Contents) > 0 {
		con.Log.Warnf("imported website metadata; %d content entries require re-upload because export does not include content bytes\n", len(exported.Contents))
	}
	return nil
}

func CloneWebsiteCmd(cmd *cobra.Command, con *core.Console) error {
	source := cmd.Flags().Arg(0)
	target := cmd.Flags().Arg(1)
	listenerID, _ := cmd.Flags().GetString("listener")
	port, _ := cmd.Flags().GetUint32("port")
	website, err := findWebsite(con, source)
	if err != nil {
		return err
	}
	web := website.GetWeb()
	if port == 0 {
		port = web.GetPort()
	}
	if listenerID == "" {
		listenerID = website.GetListenerId()
	}
	var tlsConfig *clientpb.TLS
	if website.GetCertName() != "" {
		tlsConfig = &clientpb.TLS{Enable: true}
	}
	if err := NewWebsite(con, target, web.GetRoot(), "", port, listenerID, website.GetCertName(), tlsConfig, web.GetAuth()); err != nil {
		return err
	}
	contents, err := listWebsiteContents(con, website)
	if err != nil {
		return err
	}
	if len(contents.GetContents()) > 0 {
		con.Log.Warnf("cloned website metadata; %d content entries require re-upload because ListWebContent does not include content bytes\n", len(contents.GetContents()))
	}
	return nil
}

func WebsiteCertCmd(cmd *cobra.Command, con *core.Console) error {
	return WebsiteTLSCmd(cmd, con)
}

func findWebsite(con *core.Console, key string) (*clientpb.Pipeline, error) {
	if con != nil && con.Pipelines != nil {
		if pipeline, ok := con.Pipelines[key]; ok && pipeline != nil && pipeline.GetWeb() != nil {
			return pipeline, nil
		}
	}
	websiteName, listenerID, _ := resolveWebsiteTarget(con, key)
	websites, err := con.Rpc.ListWebsites(con.Context(), &clientpb.Listener{Id: listenerID})
	if err != nil {
		return nil, err
	}
	for _, pipeline := range websites.GetPipelines() {
		if pipeline.GetName() == websiteName && pipeline.GetWeb() != nil && (listenerID == "" || pipeline.GetListenerId() == listenerID) {
			return pipeline, nil
		}
	}
	return nil, fmt.Errorf("website %s not found", key)
}

func listWebsiteContents(con *core.Console, pipeline *clientpb.Pipeline) (*clientpb.WebContents, error) {
	return con.Rpc.ListWebContent(con.Context(), &clientpb.Website{
		Name:       pipeline.GetName(),
		ListenerId: pipeline.GetListenerId(),
	})
}

func websiteToExport(pipeline *clientpb.Pipeline, contents *clientpb.WebContents) websiteExportFile {
	web := pipeline.GetWeb()
	exported := websiteExportFile{
		Name:       pipeline.GetName(),
		ListenerID: pipeline.GetListenerId(),
		Port:       web.GetPort(),
		Root:       web.GetRoot(),
		Auth:       web.GetAuth(),
		Enable:     pipeline.GetEnable(),
		CertName:   pipeline.GetCertName(),
	}
	for _, content := range contents.GetContents() {
		exported.Contents = append(exported.Contents, websiteContentItem{
			ID:          content.GetId(),
			Path:        content.GetPath(),
			Name:        content.GetName(),
			ContentType: content.GetContentType(),
			Size:        content.GetSize(),
			Comment:     content.GetComment(),
			Auth:        content.GetAuth(),
		})
	}
	return exported
}
