package cb

import (
	"time"

	"github.com/atotto/clipboard"
)

type ClipboardProvider interface {
	Clear() error
	Read() (string, error)
	Write(text string) error
}

type Clipboard struct{}

func NewClipboard() ClipboardProvider {
	return &Clipboard{}
}

func (c *Clipboard) Clear() error {
	return clipboard.WriteAll("")
}

// ReadFromClipboard reads from clipboard
// it will wait until clipboard has content
func (c *Clipboard) Read() (string, error) {
	for {
		content, err := clipboard.ReadAll()
		if err != nil {
			return "", err
		}
		if content != "" {
			return content, nil
		}
		time.Sleep(100 * time.Millisecond)
	}
}

func (c *Clipboard) Write(text string) error {
	return clipboard.WriteAll(text)
}
