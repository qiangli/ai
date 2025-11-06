package agent

import (
	"context"
	"io"
	"strings"
	"testing"

	"github.com/qiangli/ai/swarm/api"
)

type MockClipboard struct {
	content string
}

func (c *MockClipboard) Clear() error {
	return nil
}

func (c *MockClipboard) Read() (string, error) {
	return c.content, nil
}

func (c *MockClipboard) Get() (string, error) {
	return c.content, nil
}

func (c *MockClipboard) Write(text string) error {
	c.content = text
	return nil
}

func (c *MockClipboard) Append(text string) error {
	c.content += text
	return nil
}

type MockEditor struct {
	content string
}

func (e *MockEditor) Launch(_ string) (string, error) {
	return e.content, nil
}

func TestGetUserInput(t *testing.T) {
	cp := &MockClipboard{content: "clipboard content"}
	ed := &MockEditor{content: "editor content"}

	tests := []struct {
		name    string
		cfg     *api.AppConfig
		stdin   string
		want    string
		wantErr bool
	}{
		// {
		// 	name: "Command Line Message",
		// 	cfg:  &api.AppConfig{Message: "hello world", Args: []string{"command", "line"}},
		// 	want: "hello world command line",
		// },
		// {
		// 	name: "Command Line Args",
		// 	cfg:  &api.AppConfig{Args: []string{"hello", "world"}},
		// 	want: "hello world",
		// },
		// {
		// 	name:    "From Stdin",
		// 	cfg:     &api.AppConfig{Stdin: true},
		// 	stdin:   "input from stdin",
		// 	want:    "input from stdin",
		// 	wantErr: false,
		// },
		// {
		// 	name:    "From Args + Stdin",
		// 	cfg:     &api.AppConfig{Stdin: true, Args: []string{"hello", "world"}},
		// 	stdin:   "input from stdin",
		// 	want:    "###\nhello world\n###\ninput from stdin",
		// 	wantErr: false,
		// },
		// {
		// 	name: "From Clipboard",
		// 	cfg:  &api.AppConfig{Clipin: true},
		// 	want: "clipboard content",
		// },
		// {
		// 	name: "From Args + Clipboard",
		// 	cfg:  &api.AppConfig{Clipin: true, Args: []string{"hello", "world"}},
		// 	want: "###\nhello world\n###\nclipboard content",
		// },
		// {
		// 	name: "From Editor",
		// 	cfg:  &api.AppConfig{Editor: "vim", Editing: true},
		// 	want: "editor content",
		// },
		// {
		// 	name:    "Stdin takes precedence over clipboard",
		// 	cfg:     &api.AppConfig{Stdin: true, Editor: "vim"},
		// 	stdin:   "input from stdin",
		// 	want:    "input from stdin",
		// 	wantErr: false,
		// },
		// {
		// 	name:    "Stdin-clipin and editing",
		// 	cfg:     &api.AppConfig{Stdin: true, Editor: "vim", Editing: true},
		// 	stdin:   "input from stdin",
		// 	want:    "editor content",
		// 	wantErr: false,
		// },
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var ctx = context.TODO()

			var reader io.Reader
			if tt.stdin != "" {
				reader = strings.NewReader(tt.stdin)
			}

			input, err := getUserInput(ctx, tt.cfg, reader, cp, ed)

			if (err != nil) != tt.wantErr {
				t.Errorf("user input error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			got := input.Query()
			if strings.TrimSpace(got) != strings.TrimSpace(tt.want) {
				t.Errorf("user input = %v, want %v", got, tt.want)
			}
		})
	}
}
