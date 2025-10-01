package atm

import (
	"context"
	"fmt"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	// log "github.com/sirupsen/logrus"

	"github.com/qiangli/ai/swarm/api"
	swarmlog "github.com/qiangli/ai/swarm/log"
	mcpcli "github.com/qiangli/ai/swarm/mcp"
)

type McpKit struct {
	token func() (string, error)
}

func (r *McpKit) callTool(ctx context.Context, vars *api.Vars, tf *api.ToolFunc, args map[string]any) (string, error) {
	var tid = tf.ID()
	swarmlog.GetLogger(ctx).Debugf("🎖️ calling MCP tool: %s with args: %+v\n", tid, args)

	if tf.Config == nil || tf.Config.Connector == nil || tf.Config.Connector.BaseUrl == "" {
		return "", fmt.Errorf("mcp not configured: %s", tid)
	}

	client := mcpcli.NewMcpClient(tf.Config.Connector)
	token, err := r.token()
	if err != nil {
		return "", err
	}
	session, err := client.Connect(ctx, token)
	if err != nil {
		return "", err
	}
	defer session.Close()

	swarmlog.GetLogger(ctx).Debugf("Connected to mcp server session ID: %s)", session.ID())

	result, err := session.CallTool(ctx, &mcp.CallToolParams{
		Name:      tf.Name,
		Arguments: args,
	})
	if err != nil {
		return "", err
	}

	for _, content := range result.Content {
		if v, ok := content.(*mcp.TextContent); ok {
			return v.Text, nil
		}
	}

	return "", fmt.Errorf("No response")
}
