package llm

import (
	"context"
	"testing"

	"github.com/qiangli/ai/internal/log"
)

func TestDetectWorkspace(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping test in short mode.")
	}

	model := &Model{
		Name:    "gpt-4o-mini",
		BaseUrl: "http://localhost:4000",
		ApiKey:  "sk-1234",
	}

	log.SetLogLevel(log.Verbose)

	tests := []struct {
		input    string
		detected bool
	}{
		// {"is test_data empty?", false},
		// {"is ./test_data empty?", true},
		// {"is test_data folder empty?", true},
	}

	for _, test := range tests {
		resp, err := DetectWorkspace(context.TODO(), model, test.input)
		if test.detected != (err == nil) {
			t.Errorf("detect workspace: %v\n%+v", err, resp)
			return
		}
		t.Logf("detect workspace: %+v\n", resp)
	}
}

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
	}

	for _, test := range tests {
		resp, err := EvaluateCommand(context.TODO(), model, test.command, test.args)
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
