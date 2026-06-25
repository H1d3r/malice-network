package tasks

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/chainreactors/IoM-go/consts"
	"github.com/chainreactors/IoM-go/proto/client/clientpb"
	"github.com/chainreactors/malice-network/client/core"
	"github.com/chainreactors/tui"
	"github.com/evertras/bubble-table/table"
	"github.com/spf13/cobra"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

func GetTasksCmd(cmd *cobra.Command, con *core.Console) error {
	session := con.GetInteractive()
	if session == nil {
		return fmt.Errorf("session is nil")
	}

	isAll, _ := cmd.Flags().GetBool("all")
	tasks, err := con.Rpc.GetTasks(session.Context(), &clientpb.TaskRequest{
		SessionId: session.SessionId,
		All:       isAll,
	})
	if err != nil {
		return err
	}
	session.Tasks = &clientpb.Tasks{Tasks: tasks.GetTasks()}
	if 0 < len(session.Tasks.GetTasks()) {
		printTasks(session.Tasks.GetTasks(), con, isAll)
	} else {
		con.Log.Info("No tasks\n")
	}
	return nil
}

func printTasks(tasks []*clientpb.Task, con *core.Console, isAll bool) {
	var rowEntries []table.Row
	var row table.Row
	tableModel := tui.NewTable([]table.Column{
		table.NewColumn("ID", "ID", 4),
		table.NewFlexColumn("Type", "Type", 1),
		table.NewColumn("Status", "Status", 15),
		table.NewColumn("cur", "Cur", 5),
		table.NewColumn("total", "Total", 5),
		table.NewColumn("callby", "Call By", 10),
		//table.NewColumn("timeout", "timeout", 8),
	}, true)
	for _, task := range tasks {
		var status string
		if task.Status != 0 {
			status = "Error"
		} else if task.Cur != task.Total {
			status = "Running"
		} else {
			status = "Complete"
		}
		row = table.NewRow(
			table.RowData{
				"ID":     task.TaskId,
				"Type":   task.Type,
				"Status": status,
				"cur":    strconv.Itoa(int(task.Cur)),
				"total":  strconv.Itoa(int(task.Total)),
				"callby": task.Callby,
				//"timeout": strconv.FormatBool(task.Timeout),
			})
		rowEntries = append(rowEntries, row)
	}

	tableModel.SetAscSort("ID")
	tableModel.SetMultiline()
	tableModel.SetRows(rowEntries)
	con.Log.Console(tableModel.View())
}

// fetchTaskByIDs 根据逗号分隔的任务ID字符串获取任务详情
func fetchTaskByID(idStr string, con *core.Console) (*clientpb.TaskContexts, error) {

	taskId, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		return nil, fmt.Errorf("invalid task ID %q: %w", idStr, err)
	}
	task := &clientpb.Task{
		SessionId: con.GetInteractive().SessionId,
		TaskId:    uint32(taskId),
		Need:      -1,
	}
	tasksContext, err := con.Rpc.GetAllTaskContent(con.GetInteractive().Context(), task)

	return tasksContext, err
}

func TaskFetchCmd(cmd *cobra.Command, con *core.Console) error {
	taskId := cmd.Flags().Arg(0)
	toFile, _ := cmd.Flags().GetBool("file")
	outputPath, _ := cmd.Flags().GetString("output")

	tasksContext, err := fetchTaskByID(taskId, con)
	if err != nil {
		// 检查是否是 NotFound 错误
		if status.Code(err) == codes.NotFound {
			// 尝试从任务列表中查找
			taskIdNum, _ := strconv.ParseUint(taskId, 10, 32)
			if task := findTaskInList(con, uint32(taskIdNum)); task != nil {
				if !task.Finished {
					return fmt.Errorf("task %s is still running, no output available yet (progress: %d/%d)",
						taskId, task.Cur, task.Total)
				}
			}
		}
		return err
	}

	sess := con.GetInteractive()

	// 如果需要输出到文件
	if toFile || outputPath != "" {
		if outputPath == "" {
			outputPath = fmt.Sprintf("task_%s.txt", taskId)
		}

		var rendered []byte
		for _, spite := range tasksContext.Spites {
			eachTask := &clientpb.TaskContext{
				Task:    tasksContext.Task,
				Session: tasksContext.Session,
				Spite:   spite,
			}
			text, err := core.RenderTaskOutput(eachTask)
			if err != nil {
				return fmt.Errorf("failed to render task output: %w", err)
			}
			rendered = append(rendered, []byte(text+"\n")...)
		}

		if err := os.WriteFile(outputPath, rendered, 0644); err != nil {
			return fmt.Errorf("failed to write file: %w", err)
		}

		con.Log.Infof("Task output saved to: %s\n", outputPath)
		return nil
	}

	// 默认输出到控制台
	for _, spite := range tasksContext.Spites {
		eachTask := &clientpb.TaskContext{
			Task:    tasksContext.Task,
			Session: tasksContext.Session,
			Spite:   spite,
		}
		core.HandlerTask(sess, sess.Log, eachTask, nil, consts.CalleeCMD, true)
	}

	return nil
}

func findTaskInList(con *core.Console, taskId uint32) *clientpb.Task {
	session := con.GetInteractive()
	tasks, err := con.Rpc.GetTasks(session.Context(), &clientpb.TaskRequest{
		SessionId: session.SessionId,
		All:       true,
	})
	if err != nil {
		return nil
	}
	for _, task := range tasks.GetTasks() {
		if task.TaskId == taskId {
			return task
		}
	}
	return nil
}

func TaskInfoCmd(cmd *cobra.Command, con *core.Console) error {
	session := con.GetInteractive()
	if session == nil {
		return fmt.Errorf("session is nil")
	}

	taskID, err := strconv.ParseUint(cmd.Flags().Arg(0), 10, 32)
	if err != nil {
		return fmt.Errorf("invalid task ID %q: %w", cmd.Flags().Arg(0), err)
	}
	includeRaw, _ := cmd.Flags().GetBool("raw")
	includeResults, _ := cmd.Flags().GetBool("results")
	jsonOutput, _ := cmd.Flags().GetBool("json")

	details, err := con.Rpc.QueryTasks(session.Context(), &clientpb.TaskQuery{
		SessionId:             session.SessionId,
		TaskIds:               []uint32{uint32(taskID)},
		PageSize:              1,
		IncludeRequestSummary: true,
		IncludeRawRequest:     includeRaw,
		IncludeResults:        includeResults,
	})
	if err != nil {
		return err
	}
	if len(details.GetTasks()) == 0 {
		return fmt.Errorf("task %d not found", taskID)
	}

	detail := details.GetTasks()[0]
	if jsonOutput {
		rendered, err := renderTaskDetailJSON(detail)
		if err != nil {
			return err
		}
		con.Log.Console(rendered + "\n")
		return nil
	}

	return printTaskDetailInfo(detail, con, includeRaw, includeResults)
}

func printTaskDetailInfo(detail *clientpb.TaskDetail, con *core.Console, includeRaw, includeResults bool) error {
	task := detail.GetTask()
	if task == nil {
		return fmt.Errorf("task detail is missing task metadata")
	}

	lines := []string{
		fmt.Sprintf("Task: %s/%d", task.GetSessionId(), task.GetTaskId()),
		fmt.Sprintf("Type: %s", task.GetType()),
		fmt.Sprintf("Command: %s", task.GetCommandSummary()),
		fmt.Sprintf("Request: size=%d sha256=%s has=%t", task.GetRequestSize(), task.GetRequestSha256(), task.GetHasRequest()),
	}
	if task.GetRequestSummary() != "" {
		lines = append(lines, "Request Summary:")
		lines = append(lines, indentText(formatJSONString(task.GetRequestSummary()), "  "))
	}
	if includeRaw && detail.GetRawRequest() != nil {
		rawJSON, err := renderProtoJSON(detail.GetRawRequest())
		if err != nil {
			return err
		}
		lines = append(lines, "Raw Request:")
		lines = append(lines, indentText(rawJSON, "  "))
	}
	if includeResults {
		lines = append(lines, fmt.Sprintf("Results: %d", len(detail.GetResults())))
		for i, result := range detail.GetResults() {
			resultJSON, err := renderProtoJSON(result)
			if err != nil {
				return err
			}
			lines = append(lines, fmt.Sprintf("Result %d:", i))
			lines = append(lines, indentText(resultJSON, "  "))
		}
	}

	con.Log.Console(strings.Join(lines, "\n") + "\n")
	return nil
}

func renderTaskDetailJSON(detail *clientpb.TaskDetail) (string, error) {
	return renderProtoJSON(detail)
}

func renderProtoJSON(message proto.Message) (string, error) {
	data, err := protojson.MarshalOptions{
		Multiline: true,
		Indent:    "  ",
	}.Marshal(message)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func formatJSONString(value string) string {
	var formatted bytes.Buffer
	if err := json.Indent(&formatted, []byte(value), "", "  "); err != nil {
		return value
	}
	return formatted.String()
}

func indentText(value, prefix string) string {
	if value == "" {
		return ""
	}
	lines := strings.Split(value, "\n")
	for i, line := range lines {
		lines[i] = prefix + line
	}
	return strings.Join(lines, "\n")
}

//func TasksCmd(ctx *grumble.Context, con *console.Console) {
//	err := con.UpdateTasks(con.GetInteractive())
//	if err != nil {
//		console.Log.Errorf("Error updating tasks: %v", err)
//		return
//	}
//	sid := con.GetInteractive().SessionId
//	Tasks, err := con.Rpc.GetTaskFiles(con.ActiveTarget.Context(), con.GetInteractive())
//	if err != nil {
//		con.SessionLog(sid).Errorf("Error getting tasks: %v", err)
//	}
//	if 0 < len(Tasks.Tasks) {
//		PrintTasks(Tasks.Tasks, con)
//	} else {
//		console.Log.Info("No sessions")
//	}
//}
