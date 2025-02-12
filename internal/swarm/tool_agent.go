package swarm

import (
	"context"
)

func transferAgent(ctx context.Context, _ *Agent, name string, props map[string]any) (string, error) {
	transferTo, err := getStrProp("agent", props)
	if err != nil {
		return "", err
	}
	return transferTo, nil
}
