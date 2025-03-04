package api

type Output struct {
	// Agent icon and name
	Display string `json:"display"`

	Content     string `json:"content"`
	ContentType string `json:"content_type"`
}
