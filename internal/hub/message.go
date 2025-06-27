package hub

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

type Payload struct {
	Content string         `json:"content"`
	Parts   []*ContentPart `json:"parts"`
}

// ContentPart is composed of either content or url
type ContentPart struct {
	ContentType string `json:"contentType"`
	Content     string `json:"content"`
	URL         string `json:"url"`
}
