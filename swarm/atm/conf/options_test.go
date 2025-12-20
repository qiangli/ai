package conf

import (
	"fmt"
	"reflect"
	"testing"

	"github.com/qiangli/ai/swarm/api"
)

func TestParseActionArgs(t *testing.T) {
	tests := []struct {
		input    []string
		expected map[string]any
		wantErr  bool
	}{
		{
			input: []string{"ai", "@example"},
			expected: map[string]any{
				"name": "example",
				"kit":  "agent",
			},
			wantErr: false,
		},
		{
			input: []string{"agent:example"},
			expected: map[string]any{
				"name": "example",
				"kit":  "agent",
			},
			wantErr: false,
		},
		{
			input: []string{"example,", "hello"},
			expected: map[string]any{
				"name":    "example",
				"kit":     "agent",
				"message": "hello",
			},
			wantErr: false,
		},
		{
			input:    []string{"ai"},
			expected: nil,
			wantErr:  true,
		},
		{
			input: []string{"/tool:example", "--format=json", "--message=hello"},
			expected: map[string]any{
				"name":    "example",
				"kit":     "tool",
				"message": "hello",
				"format":  "json",
			},
			wantErr: false,
		},
	}

	for i, tt := range tests {
		t.Run(fmt.Sprintf("Test Case %d", i), func(t *testing.T) {
			got, err := ParseActionArgs(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseActionArgs() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			// t.Logf("got: %+v, err: %v", got, err)
			if !reflect.DeepEqual(got, api.ArgMap(tt.expected)) {
				t.Errorf("ParseActionArgs() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestParseActionCommand(t *testing.T) {
	tests := []struct {
		input    string
		expected map[string]any
		wantErr  bool
	}{
		{
			input: "ai @example",
			expected: map[string]any{
				"name": "example",
				"kit":  "agent",
			},
			wantErr: false,
		},
		{
			input: "agent:example",
			expected: map[string]any{
				"name": "example",
				"kit":  "agent",
			},
			wantErr: false,
		},
		{
			input: "example, hello",
			expected: map[string]any{
				"name":    "example",
				"kit":     "agent",
				"message": "hello",
			},
			wantErr: false,
		},
		{
			input:    "ai",
			expected: nil,
			wantErr:  true,
		},
		{
			input: "/tool:example --format=json --message=hello",
			expected: map[string]any{
				"name":    "example",
				"kit":     "tool",
				"message": "hello",
				"format":  "json",
			},
			wantErr: false,
		},
		// arg slice
		{
			input: `/tool:example --option format=json --option message=hello`,
			expected: map[string]any{
				"name":    "example",
				"kit":     "tool",
				"message": "hello",
				"format":  "json",
			},
			wantErr: false,
		},
		// arguments json object
		{
			input: `/tool:example --arguments {\"format\":\"json\",\"message\":\"hello\"}`,
			expected: map[string]any{
				"name":    "example",
				"kit":     "tool",
				"message": "hello",
				"format":  "json",
			},
			wantErr: false,
		},
		// arguments json array
		{
			input: `/tool:example --arguments [\"format=json\",\"message=hello\"]`,
			expected: map[string]any{
				"name":      "example",
				"kit":       "tool",
				"arguments": []string{"format=json", "message=hello"},
				// "message": "hello",
				// "format":  "json",
			},
			wantErr: false,
		},
		{
			input: `/tool:example --arguments [\"--format\",\"json\",\"--message\",\"hello\"]`,
			expected: map[string]any{
				"name":      "example",
				"kit":       "tool",
				"arguments": []string{"--format", "json", "--message", "hello"},
				// "message": "hello",
				// "format":  "json",
			},
			wantErr: false,
		},
		// arguments string
		{
			input: `/tool:example --arguments "format=json message=hello"`,
			expected: map[string]any{
				"name":      "example",
				"kit":       "tool",
				"arguments": []string{"format=json", "message=hello"},
				// "message": "hello",
				// "format":  "json",
			},
			wantErr: false,
		},
		{
			input: `/tool:example --arguments "--format json --message hello"`,
			expected: map[string]any{
				"name":      "example",
				"kit":       "tool",
				"arguments": []string{"--format", "json", "--message", "hello"},
				// "message": "hello",
				// "format":  "json",
			},
			wantErr: false,
		},
	}

	for i, tt := range tests {
		t.Run(fmt.Sprintf("Test Case %d", i), func(t *testing.T) {
			got, err := ParseActionCommand(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseActionArgs() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			// t.Logf("got: %+v, err: %v", got, err)
			if !reflect.DeepEqual(got, api.ArgMap(tt.expected)) {
				t.Errorf("ParseActionArgs() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestIsAction(t *testing.T) {
	tests := []struct {
		input    string
		expected bool
	}{
		{
			input:    "ai @example",
			expected: true,
		},
		{
			input:    "agent:example",
			expected: true,
		},
		{
			input:    "example, hello",
			expected: true,
		},
		{
			input:    "ai",
			expected: true,
		},
		{
			input:    "/tool:example --format=json --message=hello",
			expected: true,
		},
		{
			input:    "hell there",
			expected: false,
		},
	}

	for i, tt := range tests {
		t.Run(fmt.Sprintf("Test Case %d", i), func(t *testing.T) {
			got := IsAction(tt.input)
			if got != tt.expected {
				t.Errorf("ParseActionArgs() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestParseArgsLoggingFlags(t *testing.T) {
	tests := []struct {
		args     []string
		expected string
	}{
		{
			args:     []string{"--quiet"},
			expected: "quiet",
		},
		{
			args:     []string{"--info"},
			expected: "info",
		},
		{
			args:     []string{"--verbose"},
			expected: "verbose",
		},
		{
			args:     []string{"--log-level", "trace"},
			expected: "trace",
		},
	}

	for _, tt := range tests {
		var argm map[string]any
		var err error
		argm, err = ParseActionArgs(tt.args)
		if err != nil {
			t.FailNow()
		}

		got := argm["log_level"].(string)
		if got != tt.expected {
			t.Errorf("For args %v; expected %v got %v", tt.args, tt.expected, got)
		}
	}
}
