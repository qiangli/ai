package llm

import (
	"testing"
)

func TestHelpAgentTools(t *testing.T) {
	tools := GetAIHelpTools()

	for _, tool := range tools {
		t.Logf("tool: %+v", tool.Function.Value.Name.Value)
	}
}
