package audit

import (
	"github.com/chainreactors/IoM-go/consts"
	"github.com/chainreactors/IoM-go/proto/client/clientpb"
	"github.com/chainreactors/IoM-go/proto/implant/implantpb"
	"github.com/chainreactors/logs"
	"github.com/chainreactors/malice-network/helper/utils/fileutils"
	"github.com/chainreactors/malice-network/server/internal/configs"
	"github.com/chainreactors/malice-network/server/internal/db"
	"github.com/gookit/config/v2"
	"google.golang.org/protobuf/proto"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
)

type taskLogFile struct {
	Name        string
	TaskID      string
	ResultIndex int
}

func AuditTaskLog(sessionID string) (*clientpb.Audits, error) {
	taskDir, err := fileutils.SafeJoin(configs.ContextPath, filepath.Join(sessionID, consts.TaskPath))
	if err != nil {
		return nil, err
	}
	requestDir, err := fileutils.SafeJoin(configs.ContextPath, filepath.Join(sessionID, consts.RequestPath))
	if err != nil {
		return nil, err
	}
	re := regexp.MustCompile(`^([0-9]+)_([0-9]+)$`)
	files, err := os.ReadDir(taskDir)
	if err != nil {
		return nil, err
	}

	audits := &clientpb.Audits{}
	session, err := db.FindSession(sessionID)
	if err != nil {
		return nil, err
	}
	taskFiles := make([]taskLogFile, 0, len(files))
	for _, file := range files {
		if file.IsDir() {
			continue
		}
		matches := re.FindStringSubmatch(file.Name())
		if matches == nil {
			continue
		}
		resultIndex, err := strconv.Atoi(matches[2])
		if err != nil {
			continue
		}
		taskFiles = append(taskFiles, taskLogFile{
			Name:        file.Name(),
			TaskID:      matches[1],
			ResultIndex: resultIndex,
		})
	}
	sort.Slice(taskFiles, func(i, j int) bool {
		left, errLeft := strconv.Atoi(taskFiles[i].TaskID)
		right, errRight := strconv.Atoi(taskFiles[j].TaskID)
		if errLeft == nil && errRight == nil && left != right {
			return left < right
		}
		if taskFiles[i].TaskID != taskFiles[j].TaskID {
			return taskFiles[i].TaskID < taskFiles[j].TaskID
		}
		return taskFiles[i].ResultIndex < taskFiles[j].ResultIndex
	})
	for _, file := range taskFiles {
		taskID := file.TaskID
		taskKey := sessionID + "-" + taskID
		task, err := db.GetTask(taskKey)
		if err != nil || task == nil {
			continue
		}
		content, err := os.ReadFile(filepath.Join(taskDir, file.Name))
		if err != nil {
			logs.Log.Errorf("Error reading file: %s", err)
			continue
		}
		spite := &implantpb.Spite{}
		err = proto.Unmarshal(content, spite)
		if err != nil {
			logs.Log.Errorf("Error unmarshalling protobuf: %s", err)
			continue
		}
		audit := &clientpb.Audit{
			Context: &clientpb.TaskContext{
				Task:    task.ToProtobuf(),
				Session: session.ToProtobuf(),
				Spite:   spite,
			},
			Command:     task.Description,
			Created:     task.Created.Format("2006-01-02 15:04:05"),
			Finished:    task.FinishTime.Format("2006-01-02 15:04:05"),
			Lasted:      task.LastTime.Format("2006-01-02 15:04:05"),
			ResultIndex: int32(file.ResultIndex),
		}
		if auditLevel := config.Int(consts.ConfigAuditLevel); auditLevel > 1 {
			requestData, err := os.ReadFile(filepath.Join(requestDir, taskID))
			if err != nil {
				logs.Log.Errorf("Error reading request file: %s", err)
			} else {
				request := &implantpb.Spite{}
				err = proto.Unmarshal(requestData, request)
				if err != nil {
					logs.Log.Errorf("Error unmarshalling protobuf: %s", err)
				} else {
					audit.Request = request
				}
			}
		}

		audits.Audit = append(audits.Audit, audit)
	}
	return audits, nil
}
