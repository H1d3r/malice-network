package common

import (
	"fmt"
	"os"

	"github.com/chainreactors/IoM-go/proto/client/clientpb"
	"github.com/chainreactors/malice-network/client/core"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

// BindOutputFlags adds --file and --output flags to a command
func BindOutputFlags(cmd *cobra.Command) {
	Bind("output", false, cmd, func(f *pflag.FlagSet) {
		f.BoolP("file", "f", false, "save output to file")
		f.StringP("output", "o", "", "output file path")
	})
}

// HandleTaskOutput waits for task completion and optionally saves to file
func HandleTaskOutput(cmd *cobra.Command, con *core.Console, task *clientpb.Task) error {
	toFile, _ := cmd.Flags().GetBool("file")
	outputPath, _ := cmd.Flags().GetString("output")

	if !toFile && outputPath == "" {
		// 正常流程：异步处理
		con.GetInteractive().Console(task, string(*con.App.Shell().Line()))
		return nil
	}

	// 需要保存文件：同步等待结果
	tasksContext, err := con.Rpc.GetAllTaskContent(con.GetInteractive().Context(), &clientpb.Task{
		SessionId: task.SessionId,
		TaskId:    task.TaskId,
		Need:      -1,
	})
	if err != nil {
		return fmt.Errorf("failed to get task content: %w", err)
	}

	if outputPath == "" {
		outputPath = fmt.Sprintf("task_%d.txt", task.TaskId)
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

	// 同时也显示到控制台
	con.GetInteractive().Console(task, string(*con.App.Shell().Line()))
	return nil
}
