package agent

import (
	"github.com/qiangli/ai/internal/bubble"
)

func SimpleEditor(title, content string) (string, bool, error) {
	result, err := bubble.Edit(title, "Enter your message here...", content)
	if err != nil {
		return "", true, err
	}
	if len(result) == 0 {
		return "", true, nil
	}
	return result, false, nil
}
