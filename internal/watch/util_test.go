package watch

import (
	"testing"
)

func TestParseFile(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping test in short mode.")
	}
	tests := []struct {
		path     string
		prefix   string
		expected string
	}{
		{"testdata/file.go", "//", "// ai @agent what is fish?"},
		{"testdata/file.md", ">", ">ai @ask what is fish?"},
		{"testdata/file.py", "#", "# ai what is fish?"},
		{"testdata/file.sh", "#", "# ai /bash what is fish?"},
		{"testdata/multi.sh", "#", "# ai /bash what is fish?"},
	}

	for _, test := range tests {
		t.Run(test.path, func(t *testing.T) {
			line, err := parseFile(test.path, test.prefix)
			if err != nil {
				t.Errorf("Error parsing file: %v", err)
			}
			if line != test.expected {
				t.Errorf("Expected %s, got %s", test.expected, line)
			}
		})
	}
}

func TestParseUserInput(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping test in short mode.")
	}
	tests := []struct {
		line   string
		prefix string
		// expected
		agent   string
		command string
		content string
	}{
		{"// ai @agent what is fish?", "//", "agent", "", "what is fish?"},
		{">ai @ask what is fish?", ">", "ask", "", "what is fish?"},
		{"# ai what is fish?", "#", "", "", "what is fish?"},
		{"# ai /bash what is fish?", "#", "script", "/bash", "what is fish?"},
	}

	for _, test := range tests {
		t.Run(test.line, func(t *testing.T) {
			in, err := parseUserInput(test.line, test.prefix)
			if err != nil {
				t.Errorf("Error parsing user input: %v", err)
			}
			if in.Agent != test.agent {
				t.Errorf("Expected agent %s, got %s", test.agent, in.Agent)
			}
			if in.Command != test.command {
				t.Errorf("Expected command %s, got %s", test.command, in.Command)
			}
			if in.Message != test.content {
				t.Errorf("Expected content %s, got %s", test.content, in.Message)
			}
		})
	}
}
