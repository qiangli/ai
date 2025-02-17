package agent

import (
	"io"
	"strings"
	"testing"

	"github.com/qiangli/ai/internal"
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

func (c *MockClipboard) Write(text string) error {
	c.content = text
	return nil
}

type MockEditor struct {
	content string
}

func (e *MockEditor) Launch() (string, error) {
	return e.content, nil
}

func TestUserInput(t *testing.T) {
	cp := &MockClipboard{content: "clipboard content"}
	ed := &MockEditor{content: "editor content"}

	tests := []struct {
		name    string
		cfg     *internal.AppConfig
		stdin   string
		want    string
		wantErr bool
	}{
		{
			name: "Command Line Args",
			cfg:  &internal.AppConfig{Args: []string{"hello", "world"}},
			want: "hello world",
		},
		{
			name:    "From Stdin",
			cfg:     &internal.AppConfig{Stdin: true},
			stdin:   "input from stdin",
			want:    "input from stdin",
			wantErr: false,
		},
		{
			name:    "From Args + Stdin",
			cfg:     &internal.AppConfig{Stdin: true, Args: []string{"hello", "world"}},
			stdin:   "input from stdin",
			want:    "###\nhello world\n###\ninput from stdin",
			wantErr: false,
		},
		{
			name: "From Clipboard",
			cfg:  &internal.AppConfig{Clipin: true},
			want: "clipboard content",
		},
		{
			name: "From Args + Clipboard",
			cfg:  &internal.AppConfig{Clipin: true, Args: []string{"hello", "world"}},
			want: "###\nhello world\n###\nclipboard content",
		},
		{
			name: "From Editor",
			cfg:  &internal.AppConfig{Editor: "vim"},
			want: "editor content",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			var reader io.Reader
			if tt.stdin != "" {
				reader = strings.NewReader(tt.stdin)
			}

			input, err := userInput(
				&internal.AppConfig{
					Args:   tt.cfg.Args,
					Stdin:  tt.cfg.Stdin,
					Clipin: tt.cfg.Clipin,
					Editor: tt.cfg.Editor,
				},
				reader,
				cp,
				ed,
			)
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
