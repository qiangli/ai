package swarm

import (
	"context"
	"testing"

	"github.com/qiangli/ai/internal/api"
	"github.com/qiangli/ai/internal/log"
	"github.com/qiangli/ai/internal/util"
)

func TestEvaluateCommand(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping test in short mode.")
	}

	model := &Model{
		Name:    "gpt-4o-mini",
		BaseUrl: "http://localhost:4000",
		ApiKey:  "sk-1234",
	}

	log.SetLogLevel(log.Verbose)

	var vars = NewVars()
	sysInfo, err := util.CollectSystemInfo()
	if err != nil {
		t.Errorf("collect system info: %v", err)
	}
	vars.Arch = sysInfo.Arch
	vars.OS = sysInfo.OS
	vars.ShellInfo = sysInfo.ShellInfo
	vars.OSInfo = sysInfo.OSInfo
	vars.UserInfo = sysInfo.UserInfo
	vars.WorkDir = sysInfo.WorkDir
	vars.Models = map[api.Level]*Model{
		api.L1: model,
		api.L2: model,
		api.L3: model,
	}

	tests := []struct {
		command string
		args    []string
		safe    bool
	}{
		// {"ls", []string{}, true},
		// {"ls", []string{"-l", "/tmp"}, true},
		// {"rm", []string{"-rf", "/tmp/test"}, false},
		// {"find", []string{"./", "-name", "*.txt"}, true},
		// {"find", []string{"/tmp/test", "-type", "f", "-name", "*.exe", "-exec", "rm", "{}", "\\;"}, false},
		// {"rg", []string{"telemet(rics|ry)?", "--with-filename", "--ignore-case", "--multiline"}, true},
	}
	// fs, err := vfs.NewVFS()
	// if err != nil {
	// 	t.Errorf("create vfs: %v", err)
	// 	return
	// }
	// vars.FS = fs

	tools, _ := ListSystemTools()
	var toolMap = make(map[string]*ToolFunc)
	for _, tool := range tools {
		toolMap[tool.ID()] = tool
	}
	vars.ToolRegistry = toolMap

	for _, test := range tests {
		resp, err := evaluateCommand(context.TODO(), vars, test.command, test.args)
		if err != nil {
			t.Errorf("evaluate command: %v\n%+v", err, resp)
			return
		}
		if resp != test.safe {
			t.Errorf("evaluate command: got %v, want %v", resp, test.safe)
			return
		}
		t.Logf("evaluate command: %+v\n", resp)
	}
}
