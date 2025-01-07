package tool

import (
	"testing"
)

func TestHelpAgentTools(t *testing.T) {

	tools := HelpAgentTools()
	if len(tools) != len(helpAgentNames) {
		t.Errorf("expected 8 tools, got %d", len(tools))
	}

	for _, tool := range tools {
		t.Logf("tool: %+v", tool.Function.Value.Name.Value)
	}
}
