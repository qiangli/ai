package api

import (
	"time"
)

type Message struct {
	// message id
	ID string `json:"id"`

	// thread id
	ChatID  string    `json:"chat_id"`
	Created time.Time `json:"created"`

	// data
	ContentType string `json:"content_type"`
	Content     string `json:"content"`

	// system | assistant | user
	Role string `json:"role"`

	// user/agent
	Sender string `json:"sender"`
}
