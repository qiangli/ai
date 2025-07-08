package api

import (
	"fmt"
	"strings"
	"time"
)

const (
	TypeHeartbeat = "heartbeat"
	TypePriate    = "private"
	TypeBroadcast = "broadcast"
	TypeHub       = "hub"
	TypeRequest   = "request"
	TypeResponse  = "response"

	//
	TypeRegister   = "register"
	TypeUnregister = "unregister"
)

type Message struct {
	Type string `json:"type"`

	ID string `json:"id"`

	Sender    string `json:"sender"`
	Recipient string `json:"recipient"`

	// request
	Action string `json:"action"`

	// response
	// reply to message ID
	Reference string `json:"reference"`

	// reply status code 100 200 400 500
	Code string `json:"code"`

	// request/response
	Payload string `json:"payload"`

	Timestamp *time.Time `json:"timestamp"`
}

func (r *Message) String() string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Type: %s\n", r.Type))
	sb.WriteString(fmt.Sprintf("ID: %s\n", r.ID))
	sb.WriteString(fmt.Sprintf("Reference: %s\n", r.Reference))
	sb.WriteString(fmt.Sprintf("Sender: %s\n", r.Sender))
	sb.WriteString(fmt.Sprintf("Recipient: %s\n", r.Recipient))
	sb.WriteString(fmt.Sprintf("Action: %s\n", r.Action))
	sb.WriteString(fmt.Sprintf("Code: %s\n", r.Code))
	sb.WriteString(fmt.Sprintf("Payload: %v bytes\n", len(r.Payload)))
	sb.WriteString(fmt.Sprintf("Timestamp: %v\n", r.Timestamp))

	return sb.String()
}

type Payload struct {
	Version string `json:"version"`

	Format   string `json:"format"`
	Messages string `json:"messages"`

	Content string         `json:"content"`
	Parts   []*ContentPart `json:"parts"`
}

// ContentPart is composed of either content or url
type ContentPart struct {
	ContentType string `json:"contentType"`
	Content     string `json:"content"`
	URL         string `json:"url"`
}
