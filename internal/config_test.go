package internal

import (
	"fmt"
	"testing"

	fangs "github.com/spf13/viper"

	"github.com/qiangli/ai/swarm/api"
)

func TestParseConfig(t *testing.T) {
	const defaultAgent = ""
	var viper *fangs.Viper = fangs.New()
	tests := []struct {
		args []string
		// agent/command
		expected     []string
		expectedArgs []string
	}{
		{
			args:         []string{},
			expected:     []string{defaultAgent, ""},
			expectedArgs: []string{},
		},
		// {
		// 	args:         []string{"/"},
		// 	expected:     []string{"shell", ""},
		// 	expectedArgs: []string{},
		// },
		// {
		// 	args:         []string{"/which"},
		// 	expected:     []string{"shell", "which"},
		// 	expectedArgs: []string{},
		// },
		// {
		// 	args:         []string{"/", "test"},
		// 	expected:     []string{"shell", ""},
		// 	expectedArgs: []string{"test"},
		// },
		// {
		// 	args:         []string{"/which", "a", "test"},
		// 	expected:     []string{"shell", "which"},
		// 	expectedArgs: []string{"a", "test"},
		// },
		{
			args:         []string{"@"},
			expected:     []string{"anonymous", ""},
			expectedArgs: []string{},
		},
		{
			args:         []string{"@agent"},
			expected:     []string{"agent", ""},
			expectedArgs: []string{},
		},
		{
			args:         []string{"@", "test"},
			expected:     []string{"anonymous", ""},
			expectedArgs: []string{"test"},
		},
		{
			args:         []string{"@agent", "a", "test"},
			expected:     []string{"agent", ""},
			expectedArgs: []string{"a", "test"},
		},
		{
			args:         []string{"test", "@"},
			expected:     []string{"anonymous", ""},
			expectedArgs: []string{"test"},
		},
		{
			args:         []string{"test", "@shell"},
			expected:     []string{"shell", ""},
			expectedArgs: []string{"test"},
		},
		// {
		// 	args:         []string{"this", "is", "a", "test", "@agent/which"},
		// 	expected:     []string{"agent", "which"},
		// 	expectedArgs: []string{"this", "is", "a", "test"},
		// },
	}

	for i, test := range tests {
		t.Run(fmt.Sprintf("test - %d", i), func(t *testing.T) {
			var cfg = &api.AppConfig{}
			err := ParseConfig(viper, cfg, test.args)
			if err != nil {
				t.Fatalf("expected no error, got %v", err)
			}

			if cfg.Name != test.expected[0] {
				t.Fatalf("expected %v, got %s", test.expected, cfg.Name)
			}

			// if cfg.Agent != test.expected[0] || cfg.Command != test.expected[1] {
			// 	t.Fatalf("expected %v, got %s %s", test.expected, cfg.Agent, cfg.Command)
			// }

			// if len(cfg.Args) != len(test.expectedArgs) {
			// 	t.Fatalf("expected args length %d, got %d", len(test.expectedArgs), len(cfg.Args))
			// }
			// for j, arg := range cfg.Args {
			// 	if arg != test.expectedArgs[j] {
			// 		t.Fatalf("expected arg %v, got %v", test.expectedArgs[j], arg)
			// 	}
			// }
		})
	}
}
