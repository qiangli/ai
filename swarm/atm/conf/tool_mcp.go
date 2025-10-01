package conf

import (
	"context"
	"fmt"

	// log "github.com/sirupsen/logrus"

	"github.com/qiangli/ai/swarm/api"
	mcpcli "github.com/qiangli/ai/swarm/mcp"
)

func listMcpTools(tc *api.ToolsConfig, token string) ([]*api.ToolFunc, error) {
	ctx := context.Background()

	if tc.Connector == nil || tc.Connector.BaseUrl == "" {
		return nil, fmt.Errorf("Invalid mcp config. Missing URL")
	}

	// log.Debugf("Connecting to MCP server at %s", tc.Connector.BaseUrl)
	client := mcpcli.NewMcpClient(tc.Connector)
	session, err := client.Connect(ctx, token)
	if err != nil {
		return nil, err
	}
	defer session.Close()

	// log.Debugf("Connected to server: session ID: %s)", session.ID())

	result, err := session.ListTools(ctx, nil)
	if err != nil {
		return nil, err
	}

	tools := make([]*api.ToolFunc, 0)
	for _, v := range result.Tools {
		params, err := structToMap(v.InputSchema)
		if err != nil {
			return nil, err
		}
		tool := &api.ToolFunc{
			Kit:         tc.Kit,
			Type:        api.ToolTypeMcp,
			Name:        v.Name,
			Description: v.Description,
			Parameters:  params,
			Body:        nil,
			//
			Provider: nvl(tc.Connector.Provider, tc.Provider),
			BaseUrl:  nvl(tc.Connector.BaseUrl, tc.BaseUrl),
			ApiKey:   nvl(tc.Connector.ApiKey, tc.ApiKey),
			//
			Config: tc,
		}

		tools = append(tools, tool)
	}

	return tools, nil
}
