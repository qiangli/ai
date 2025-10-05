package atm

import (
	"context"
	"os"
	"testing"

	"github.com/qiangli/ai/swarm/api"
	"github.com/qiangli/ai/swarm/log"
	"github.com/qiangli/ai/swarm/vos"
)

func TestEvaluateCommand(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping test in short mode.")
	}

	m := &api.Model{
		Model:    "gpt-5-nano",
		BaseUrl:  "https://api.openai.com/v1",
		ApiKey:   os.Getenv("OPENAI_API_KEY"),
		Provider: "openai",
	}

	var ctx = context.WithValue(context.TODO(), ModelsContextKey, m)

	log.GetLogger(ctx).SetLogLevel(log.Verbose)

	var vars = api.NewVars()
	vars.Config = &api.AppConfig{}

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
		// {"find", []string{"./", "-type", "f", "|", "xargs", "grep", "-l", "xyz"}, true},
		//
		// {"find", []string{"./", "-type", "f", "-name", "*.yaml", "-exec", "awk", "/items:/{if(!match($0,/^type: array/)){print FILENAME}}", "{}", "+", "|", "sort", "-u"}, true},
		{"find", []string{"./", "-name", "*.sql", "-exec", "grep", "-l", "s3_files", "{}", "\\;"}, true},
	}

	vs := vos.NewLocalSystem()

	for _, test := range tests {
		resp, err := EvaluateCommand(ctx, vs, vars, test.command, test.args)
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
