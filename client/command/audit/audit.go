package audit

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/chainreactors/IoM-go/client"
	"github.com/chainreactors/IoM-go/proto/client/clientpb"
	"github.com/chainreactors/logs"
	"github.com/chainreactors/malice-network/client/assets"
	"github.com/chainreactors/malice-network/client/core"
	"github.com/chainreactors/malice-network/helper/intermediate"
	"github.com/spf13/cobra"
	"html/template"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// AuditExport 用于导出 JSON
// 保证字段顺序和命名符合需求
// 注意 taskResult 保留
type AuditExport struct {
	SessionID      string      `json:"session"`
	TaskID         string      `json:"task"`
	Type           string      `json:"type"`
	Status         int32       `json:"status"`
	Command        string      `json:"command"`
	CommandSummary string      `json:"commandSummary"`
	CallBy         string      `json:"callby"`
	Total          int32       `json:"total"`
	Cur            int32       `json:"cur"`
	TaskFinished   bool        `json:"taskFinished"`
	Timeout        bool        `json:"timeout"`
	Created        string      `json:"created"`
	Finished       string      `json:"finished"`
	Lasted         string      `json:"lasted"`
	CreatedAt      int64       `json:"createdAt"`
	FinishedAt     int64       `json:"finishedAt"`
	RequestSummary string      `json:"requestSummary"`
	RequestSize    int64       `json:"requestSize"`
	RequestSHA256  string      `json:"requestSha256"`
	HasRequest     bool        `json:"hasRequest"`
	ResultIndex    int32       `json:"resultIndex"`
	Response       interface{} `json:"response"`
	Request        interface{} `json:"request"`
	TaskResult     string      `json:"taskResult"`
}

func AuditSessionCmd(cmd *cobra.Command, con *core.Console) error {
	sessionID := cmd.Flags().Arg(0)
	if sessionID == "" {
		return fmt.Errorf("session id is required")
	}
	output, _ := cmd.Flags().GetString("output")
	path, _ := cmd.Flags().GetString("file")
	ext := strings.ToLower(output)
	var isJson bool
	var format string
	switch ext {
	case "json":
		isJson = true
		format = ".json"
	case "html", "htm":
		isJson = false
		format = ".html"
	default:
		return fmt.Errorf("unsupported export format: %s", ext)
	}
	auditLog, err := con.Rpc.GetAudit(con.Context(), &clientpb.SessionRequest{
		SessionId: sessionID,
	})
	if err != nil {
		return err
	}
	if path == "" {
		path = filepath.Join(assets.GetTempDir(), sessionID+format)
	}

	if isJson {
		// 组装导出结构体
		var exportList []AuditExport
		for _, a := range auditLog.Audit {
			if a == nil || a.Context == nil || a.Context.Task == nil || a.Context.Session == nil {
				continue
			}
			task := a.Context.Task
			taskResult := renderAuditTaskResult(a.Context)
			exportList = append(exportList, AuditExport{
				SessionID:      a.Context.Session.SessionId,
				TaskID:         strconv.Itoa(int(task.TaskId)),
				Type:           task.Type,
				Status:         task.Status,
				Total:          task.Total,
				Cur:            task.Cur,
				Command:        a.Command,
				CommandSummary: task.CommandSummary,
				CallBy:         task.Callby,
				TaskFinished:   task.Finished,
				Timeout:        task.Timeout,
				Created:        a.Created,
				Finished:       a.Finished,
				Lasted:         a.Lasted,
				CreatedAt:      task.CreatedAt,
				FinishedAt:     task.FinishedAt,
				RequestSummary: task.RequestSummary,
				RequestSize:    task.RequestSize,
				RequestSHA256:  task.RequestSha256,
				HasRequest:     task.HasRequest,
				ResultIndex:    a.ResultIndex,
				Response:       a.Context.Spite,
				Request:        a.Request,
				TaskResult:     taskResult,
			})
		}
		data, err := json.MarshalIndent(exportList, "", "  ")
		if err != nil {
			return err
		}
		err = os.WriteFile(path, data, 0644)
		if err != nil {
			return err
		}
		con.Log.Infof("%s audit log saved at %s\n", sessionID, path)
		return nil
	}

	// HTML 渲染
	data, err := renderAuditHTML(auditLog.Audit)
	if err != nil {
		return err
	}
	err = os.WriteFile(path, data, 0644)
	if err != nil {
		return err
	}
	con.Log.Infof("%s audit log saved at %s\n", sessionID, path)
	return nil
}

func renderAuditTaskResult(ctx *clientpb.TaskContext) string {
	if ctx == nil || ctx.Task == nil {
		return "No task result available"
	}
	fn, ok := intermediate.InternalFunctions[ctx.Task.Type]
	if !ok || fn.FinishCallback == nil {
		return "No task result available"
	}
	resp, err := fn.FinishCallback(ctx)
	if err != nil {
		logs.Log.Errorf("failed to parse task: %s", err)
		return fmt.Sprintf("Error parsing task: %s", err.Error())
	}
	return client.RemoveANSI(resp)
}

// renderAuditHTML
func renderAuditHTML(entries []*clientpb.Audit) ([]byte, error) {
	type AuditView struct {
		*clientpb.Audit
		RequestOmitted bool
		TaskResult     string
	}
	var auditsView []AuditView
	for _, a := range entries {
		if a == nil || a.Context == nil || a.Context.Task == nil {
			continue
		}
		reqBytes, _ := json.Marshal(a.Request)
		audit := AuditView{
			Audit:          a,
			RequestOmitted: len(reqBytes) > 100*1024,
		}
		audit.TaskResult = renderAuditTaskResult(a.Context)
		auditsView = append(auditsView, audit)
	}

	funcMap := template.FuncMap{
		"formatjson": func(v interface{}) string {
			b, _ := json.MarshalIndent(v, "", "  ")
			return string(b)
		},
		"formatTaskInfo": func(audit *clientpb.Audit) string {
			task := audit.Context.Task
			jsonStr := fmt.Sprintf(`{
  "sessionId": "%s",
  "taskId": %d,
  "resultIndex": %d,
  "command": "%s",
  "commandSummary": "%s",
  "created": "%s",
  "finished": "%s",
  "lasted": "%s",
  "createdAt": %d,
  "finishedAt": %d,
  "taskType": "%s",
  "taskStatus": %d,
  "callby": "%s",
  "total": %d,
  "cur": %d,
  "taskFinished": %t,
  "timeout": %t,
  "requestSummary": "%s",
  "requestSize": %d,
  "requestSha256": "%s",
  "hasRequest": %t
}`,
				audit.Context.Session.SessionId,
				task.TaskId,
				audit.ResultIndex,
				template.JSEscapeString(audit.Command),
				template.JSEscapeString(task.CommandSummary),
				audit.Created,
				audit.Finished,
				audit.Lasted,
				task.CreatedAt,
				task.FinishedAt,
				task.Type,
				task.Status,
				template.JSEscapeString(task.Callby),
				task.Total,
				task.Cur,
				task.Finished,
				task.Timeout,
				template.JSEscapeString(task.RequestSummary),
				task.RequestSize,
				template.JSEscapeString(task.RequestSha256),
				task.HasRequest,
			)
			return jsonStr
		},
		"len": func(v interface{}) int {
			switch val := v.(type) {
			case []AuditView:
				return len(val)
			default:
				return 0
			}
		},
		"js": func(s string) string {
			return template.JSEscapeString(s)
		},
	}

	data := struct {
		Entries       []AuditView
		GeneratedTime string
	}{
		Entries:       auditsView,
		GeneratedTime: time.Now().Format("2006-01-02 15:04:05"),
	}

	var buf bytes.Buffer
	t := template.Must(template.New("audit").Funcs(funcMap).Parse(string(assets.AuditHtml)))
	err := t.Execute(&buf, data)
	if err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}
