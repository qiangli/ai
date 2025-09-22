package swarm

// import (
// 	"context"
// 	"testing"

// 	"github.com/qiangli/ai/swarm/log"
// 	"github.com/qiangli/ai/internal/util"
// 	"github.com/qiangli/ai/swarm/api"
// 	// "github.com/qiangli/ai/swarm/llm"
// )

// func TestEvaluateCommand(t *testing.T) {
// 	if testing.Short() {
// 		t.Skip("skipping test in short mode.")
// 	}

// 	m := &api.Model{
// 		Model:   "gpt-4o-mini",
// 		BaseUrl: "http://localhost:4000",
// 		ApiKey:  "sk-1234",
// 	}

// 	log.GetLogger(ctx).SetLogLevel(log.Verbose)

// 	var vars = api.NewVars()
// 	vars.Config = &api.AppConfig{
// 		ModelLoader: func(level string) (*api.Model, error) {
// 			return m, nil
// 		},
// 	}
// 	sysInfo, err := util.CollectSystemInfo()
// 	if err != nil {
// 		t.Errorf("collect system info: %v", err)
// 	}
// 	vars.Arch = sysInfo.Arch
// 	vars.OS = sysInfo.OS
// 	vars.ShellInfo = sysInfo.ShellInfo
// 	vars.OSInfo = sysInfo.OSInfo
// 	vars.UserInfo = sysInfo.UserInfo
// 	// vars.WorkDir = sysInfo.WorkDir
// 	// vars.Models = map[model.Level]*model.Model{
// 	// 	model.L1: m,
// 	// 	model.L2: m,
// 	// 	model.L3: m,
// 	// }

// 	tests := []struct {
// 		command string
// 		args    []string
// 		safe    bool
// 	}{
// 		// {"ls", []string{}, true},
// 		// {"ls", []string{"-l", "/tmp"}, true},
// 		// {"rm", []string{"-rf", "/tmp/test"}, false},
// 		// {"find", []string{"./", "-name", "*.txt"}, true},
// 		// {"find", []string{"/tmp/test", "-type", "f", "-name", "*.exe", "-exec", "rm", "{}", "\\;"}, false},
// 		// {"rg", []string{"telemet(rics|ry)?", "--with-filename", "--ignore-case", "--multiline"}, true},
// 		// {"find", []string{"./", "-type", "f", "|", "xargs", "grep", "-l", "xyz"}, true},
// 		// wrong
// 		// {"find", []string{"./", "-type", "f", "-name", "*.yaml", "-exec", "awk", "/items:/{if(!match($0,/^type: array/)){print FILENAME}}", "{}", "+", "|", "sort", "-u"}, false},
// 		{"find", []string{"./", "-name", "*.sql", "-exec", "grep", "-l", "s3_files", "{}", "\\;"}, true},
// 	}

// 	// tools, err := listTools(&api.AppConfig{})
// 	// if err != nil {
// 	// 	t.Errorf("list tools: %v", err)
// 	// }
// 	// var toolMap = make(map[string]*api.ToolFunc)
// 	// for _, tool := range tools {
// 	// 	toolMap[tool.ID()] = tool
// 	// }
// 	// vars.ToolRegistry = toolMap

// 	for _, test := range tests {
// 		resp, err := evaluateCommand(context.TODO(), vars, test.command, test.args)
// 		if err != nil {
// 			t.Errorf("evaluate command: %v\n%+v", err, resp)
// 			return
// 		}
// 		if resp != test.safe {
// 			t.Errorf("evaluate command: got %v, want %v", resp, test.safe)
// 			return
// 		}
// 		t.Logf("evaluate command: %+v\n", resp)
// 	}
// }
