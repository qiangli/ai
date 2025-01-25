package resource

import (
	"strings"
	"testing"

	"github.com/qiangli/ai/internal/resource/pr"
)

func TestGetPrDescriptionSystem(t *testing.T) {
	tests := []struct {
		name     string
		expected []string
	}{
		{
			name: "test",
			expected: []string{
				`"$schema": "http://json-schema.org/draft-07/schema#",`,
				`"type": ["Enhancement", "Documentation"],`,
			},
		},
	}

	contains := func(text string, want string) bool {
		return strings.Contains(text, want)
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			text, err := getPrDescriptionSystem()
			if err != nil {
				t.Errorf("GetPrDescriptionSystem: %v", err)
			}
			for _, want := range tt.expected {
				if !contains(text, want) {
					t.Errorf("GetPrDescriptionSystem: got %v, want %v", text, want)
				}
			}
			t.Logf("TestGetPrDescriptionSystem:\n%v", text)
		})
	}
}

func TestGetPrUser(t *testing.T) {
	tests := []struct {
		name      string
		msg       string
		diff      string
		changelog string

		expected []bool
	}{
		{
			name:      "test1 - input",
			msg:       "my input",
			diff:      "my diff",
			changelog: "",
			expected:  []bool{true, true, false},
		},
		{
			name:      "test2 - no input",
			msg:       "",
			diff:      "my diff",
			changelog: "",
			expected:  []bool{false, true, false},
		},
		{
			name:      "test3 - change log",
			msg:       "my input",
			diff:      "my diff",
			changelog: "my changelog",
			expected:  []bool{true, true, true},
		},
	}

	contains := func(text string, want string) bool {
		return strings.Contains(text, want)
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			txt, err := GetPrUser(&pr.Input{
				Instruction: tt.msg,
				Diff:        tt.diff,
				ChangeLog:   tt.changelog,
			})
			if err != nil {
				t.Errorf("TestGetPrUser: %v", err)
			}
			if contains(txt, "my input") != tt.expected[0] {
				t.Errorf("TestGetPrUser: got %v, want %v", txt, "my input")
			}
			if contains(txt, "my diff") != tt.expected[1] {
				t.Errorf("TestGetPrUser: got %v, want %v", txt, "my diff")
			}
			if contains(txt, "my changelog") != tt.expected[2] {
				t.Errorf("TestGetPrUser: got %v, want %v", txt, "my changelog")
			}

			t.Logf("TestGetPrUser:\n%s\n%v", tt.name, txt)
		})
	}
}

func TestFormatPrDescription(t *testing.T) {
	out, err := formatPrDescription(prDescrptionExample)
	if err != nil {
		t.Errorf("FormatPrDescription: %v", err)
	}
	t.Logf("TestFormatPrDescription:\n%v", out)
}

func TestGetPrReviewSystem(t *testing.T) {
	tests := []struct {
		name string
	}{
		{name: "test"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			text, err := getPrReviewSystem()
			if err != nil {
				t.Errorf("TestGetPrReviewSystem: %v", err)
			}
			t.Logf("TestGetPrReviewSystem:\n%v", text)
		})
	}
}

func TestGetPrCodeSystem(t *testing.T) {
	tests := []struct {
		name string
	}{
		{name: "test"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			text, err := getPrCodeSystem()
			if err != nil {
				t.Errorf("TestGetPrCodeSystem: %v", err)
			}
			t.Logf("TestGetPrCodeSystem:\n%v", text)
		})
	}
}
