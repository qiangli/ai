package util

import (
	"encoding/json"

	"github.com/qiangli/ai/swarm/api"
)

func FormatContent(format string, out *api.Output) (string, error) {
	switch format {
	case "markdown":
		// TODO: markdown formatting lost if the content is also tee'd to a file
		return Render(out.Content), nil
	case "json":
		obj, err := json.Marshal(out)
		if err != nil {
			return "", err
		}
		return string(obj), nil
	case "text":
		return out.Content, nil
	default:
		return out.Content, nil
	}
}
