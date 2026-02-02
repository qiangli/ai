package util

import (
	"time"

	"github.com/atotto/clipboard"
)

type Clipboard struct{}

func NewClipboard() *Clipboard {
	return &Clipboard{}
}

func (c *Clipboard) Clear() error {
	return clipboard.WriteAll("")
}

// Read reads from clipboard
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

// Get tries to read from clipboard
// it will return empty string if clipboard is empty
func (c *Clipboard) Get() (string, error) {
	return clipboard.ReadAll()
}

func (c *Clipboard) Write(text string) error {
	return clipboard.WriteAll(text)
}

func (c *Clipboard) Append(text string) error {
	old, _ := clipboard.ReadAll()
	return clipboard.WriteAll(old + text)
}
