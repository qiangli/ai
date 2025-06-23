package server

import (
	"time"
)

type Message struct {
	ID        string    `json:"id"`
	Type      string    `json:"type"`
	Sender    string    `json:"sender"`
	Recipient string    `json:"recipient"`
	Payload   string    `json:"payload"`
	Timestamp time.Time `json:"timestamp"`
}
