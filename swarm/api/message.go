package api

import (
	"time"
)

type Message struct {
	ID string `json:"id"`

	// thread
	ChatID  string    `json:"chatId"`
	Created time.Time `json:"created"`

	// data
	ContentType string `json:"contentType"`
	Content     string `json:"content"`

	// system | assistant | user
	Role string `json:"role"`

	// user/agent
	Sender string `json:"sender"`
}
