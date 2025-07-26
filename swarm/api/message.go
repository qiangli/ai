package api

import (
	"time"
)

type Message struct {
	ID      string    `json:"id"`
	ChatID  string    `json:"chatId"`
	Created time.Time `json:"created"`

	// data
	ContentType string `json:"contentType"`
	Content     string `json:"content"`

	Role string `json:"role"`

	// agent name
	Sender string `json:"sender"`

	// model alias
	Models string `json:"models"`
}
