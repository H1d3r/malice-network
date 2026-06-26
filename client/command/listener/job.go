package listener

import (
	"errors"
	"github.com/chainreactors/malice-network/client/core"
	"strconv"
	"strings"

	"github.com/chainreactors/IoM-go/proto/client/clientpb"
	"github.com/chainreactors/tui"
	"github.com/evertras/bubble-table/table"
	"github.com/spf13/cobra"
)

func ListJobsCmd(cmd *cobra.Command, con *core.Console) error {
	Pipelines, err := con.Rpc.ListJobs(con.Context(), &clientpb.Empty{})
	if err != nil {
		return err
	}
	if len(Pipelines.GetPipelines()) == 0 {
		con.Log.Importantf("No jobs found")
		return nil
	}
	var rowEntries []table.Row
	var row table.Row
	tableModel := tui.NewTable([]table.Column{
		table.NewFlexColumn("Name", "Name", 1),
		table.NewColumn("Listener", "Listener", 15),
		table.NewColumn("IP", "IP", 16),
		table.NewColumn("Port", "Port", 7),
		table.NewColumn("Type", "Type", 7),
	}, true)
	for _, pipeline := range Pipelines.GetPipelines() {
		switch pipeline.Body.(type) {
		case *clientpb.Pipeline_Tcp:
			tcp := pipeline.GetTcp()
			row = table.NewRow(
				table.RowData{
					"Name":     pipeline.Name,
					"Listener": pipeline.ListenerId,
					"IP":       pipeline.Ip,
					"Port":     strconv.Itoa(int(tcp.Port)),
					"Type":     "TCP",
				})
		case *clientpb.Pipeline_Web:
			website := pipeline.GetWeb()
			row = table.NewRow(
				table.RowData{
					"Name":     pipeline.Name,
					"Listener": pipeline.ListenerId,
					"IP":       pipeline.Ip,
					"Port":     strconv.Itoa(int(website.Port)),
					"Type":     "Web",
				})
		default:
			row = table.NewRow(table.RowData{
				"Name":     pipeline.Name,
				"Listener": pipeline.ListenerId,
				"IP":       pipeline.Ip,
				"Port":     "",
				"Type":     pipeline.Type,
			})
		}
		rowEntries = append(rowEntries, row)
	}
	tableModel.SetMultiline()
	tableModel.SetRows(rowEntries)
	con.Log.Console(tableModel.View())
	return nil
}

func InspectJobCmd(cmd *cobra.Command, con *core.Console) error {
	name := cmd.Flags().Arg(0)
	pipeline, err := findJob(con, name)
	if err != nil {
		return err
	}
	printPipelineDetail(pipeline)
	return nil
}

func KillJobCmd(cmd *cobra.Command, con *core.Console) error {
	name := cmd.Flags().Arg(0)
	pipeline, err := findJob(con, name)
	if err != nil {
		return err
	}
	_, err = con.Rpc.StopPipeline(con.Context(), &clientpb.CtrlPipeline{
		Name:       pipeline.GetName(),
		ListenerId: pipeline.GetListenerId(),
	})
	return err
}

func findJob(con *core.Console, key string) (*clientpb.Pipeline, error) {
	listenerID, name, ok := strings.Cut(key, ":")
	if !ok {
		name = key
		listenerID = ""
	}
	jobs, err := con.Rpc.ListJobs(con.Context(), &clientpb.Empty{})
	if err != nil {
		return nil, err
	}
	for _, pipeline := range jobs.GetPipelines() {
		if pipeline.GetName() == name && (listenerID == "" || pipeline.GetListenerId() == listenerID) {
			return pipeline, nil
		}
	}
	return nil, errors.New("job not found")
}
