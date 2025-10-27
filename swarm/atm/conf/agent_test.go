package conf

import (
	"io"
	"os"
	"testing"
)

func TestLoadAgentData(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping test in short mode.")
	}
	// file, err := os.Open("../resource/incubator/agents/gptr/agent.yaml")
	// file, err := os.Open("../resource/incubator/agents/think/agent.yaml")
	file, err := os.Open("../resource/incubator/agents/cron/agent.yaml")
	if err != nil {
		t.Fatalf("failed to open file: %v", err)
	}
	defer file.Close()

	d, err := io.ReadAll(file)
	if err != nil {
		t.Fatalf("failed to read file: %v", err)
	}

	ac, err := LoadAgentsData([][]byte{d})
	if err != nil {
		t.Fatalf("failed to load file: %v", err)
	}
	for _, v := range ac.Agents {
		t.Logf("%+v", v)
	}
}
