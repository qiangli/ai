package hub

import (
	"testing"

	"github.com/qiangli/ai/swarm/api"
)

func TestParser(t *testing.T) {
	cfg := &api.AppConfig{
		Agent:   "swe",
		Command: "",
	}
	tests := []struct {
		input string
	}{
		{"ai -n --agent code --models anthropic convert the image to html/css code"},
		{"ai tell me a joke --agent swe -m anthropic"},
		{"ai --agent git/short --models=anthropic --new --max-history=100 --unsafe --max-turns 1 this is a test"},
	}

	for _, tc := range tests {
		if line, ok := triggerAI(tc.input); ok {
			err := parseFlags(line, cfg)
			if err != nil {
				t.Fatal(err)
			}
			t.Logf("cfg: %+v", cfg)
			continue
		}
		t.Fatal("ai not triggered")
	}
}
