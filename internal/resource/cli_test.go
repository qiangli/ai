package resource

import (
	"strings"
	"testing"
)

func TestGetCliConfigSystem(t *testing.T) {
	tests := []struct {
		name     string
		expected []string
	}{
		{
			name: "test",
			expected: []string{
				`"$schema": "http://json-schema.org/draft-07/schema#",`,
			},
		},
	}

	contains := func(text string, want string) bool {
		return strings.Contains(text, want)
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			text, err := GetCliConfigSystem()
			if err != nil {
				t.Errorf("GetCliConfigSystem: %v", err)
			}
			for _, want := range tt.expected {
				if !contains(text, want) {
					t.Errorf("GetCliConfigSystem: got %v, want %v", text, want)
				}
			}
			t.Logf("TestGetCliConfigSystem:\n%v", text)
		})
	}
}

func TestGetCliConfigUser(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []string
	}{
		{
			name:  "test",
			input: "my input",
			expected: []string{
				`my input`,
			},
		},
	}

	contains := func(text string, want string) bool {
		return strings.Contains(text, want)
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			text, err := GetCliConfigUser(tt.input)
			if err != nil {
				t.Errorf("GetCliConfigUser: %v", err)
			}
			for _, want := range tt.expected {
				if !contains(text, want) {
					t.Errorf("GetCliConfigUser: got %v, want %v", text, want)
				}
			}
			t.Logf("TestGetCliConfigUser:\n%v", text)
		})
	}
}

func TestGetCliGptrReportSystem(t *testing.T) {
	tests := []struct {
		name        string
		reportTypes map[string]string
		tones       map[string]string
	}{
		{
			name: "test",
			reportTypes: map[string]string{
				"type1": "type 1",
				"type2": "type 2",
			},
			tones: map[string]string{
				"tone1": "tone 1",
				"tone2": "tone 2",
				"tone3": "tone 3",
			},
		},
	}

	contains := func(text string, want string) bool {
		return strings.Contains(text, want)
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			text, err := GetCliGptrReportSystem(tt.reportTypes, tt.tones)
			if err != nil {
				t.Errorf("GetCliGptrReportSystem: %v", err)
			}
			for _, want := range tt.reportTypes {
				if !contains(text, want) {
					t.Errorf("GetCliGptrReportSystem: got %v, want %v", text, want)
				}
			}
			for _, want := range tt.tones {
				if !contains(text, want) {
					t.Errorf("GetCliGptrReportSystem: got %v, want %v", text, want)
				}
			}
			t.Logf("TestGetCliGptrReportSystem:\n%v", text)
		})
	}
}
