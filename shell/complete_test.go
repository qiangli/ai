package shell

import (
	"reflect"
	"testing"

	"github.com/c-bata/go-prompt"
)

func TestUnique(t *testing.T) {
	tests := []struct {
		name     string
		input    []prompt.Suggest
		expected []prompt.Suggest
	}{
		{
			name: "No duplicates",
			input: []prompt.Suggest{
				{Text: "one", Description: "First"},
				{Text: "two", Description: "Second"},
			},
			expected: []prompt.Suggest{
				{Text: "one", Description: "First"},
				{Text: "two", Description: "Second"},
			},
		},
		{
			name: "With duplicates",
			input: []prompt.Suggest{
				{Text: "one", Description: "First"},
				{Text: "one", Description: "First"},
				{Text: "two", Description: "Second"},
			},
			expected: []prompt.Suggest{
				{Text: "one", Description: "First"},
				{Text: "two", Description: "Second"},
			},
		},
		{
			name:     "Empty input",
			input:    []prompt.Suggest{},
			expected: []prompt.Suggest{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := unique(tt.input)
			if !reflect.DeepEqual(result, tt.expected) {
				t.Errorf("got %v, want %v", result, tt.expected)
			}
		})
	}
}
