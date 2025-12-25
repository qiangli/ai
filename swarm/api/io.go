package api

const (
	ContentTypeImageB64 = "img/base64"
)

type ClipboardProvider interface {
	Clear() error
	Read() (string, error)
	Get() (string, error)
	Write(string) error
	Append(string) error
}

type EditorProvider interface {
	Launch(string) (string, error)
}

type UserInput struct {
	// cached media contents
	Messages []*Message `json:"-"`

	Message string
}

type Output struct {
	// agent emoji and name
	Display string `json:"display"`

	Content     string `json:"content"`
	ContentType string `json:"content_type"`
}

// type IOFilter struct {
// 	Agent string
// }

// // https://modelcontextprotocol.io/specification/2025-06-18/client/elicitation
// type Elicitation struct {
// 	Method string
// 	Params map[string]any
// 	Result map[string]any
// }
