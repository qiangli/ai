package conf

import (
	"context"
	"fmt"
	"maps"

	"github.com/qiangli/ai/swarm/api"
	mcpcli "github.com/qiangli/ai/swarm/mcp"
)

func listMcpTools(kit string, tc *api.ToolConfig, token string) ([]*api.ToolFunc, error) {
	ctx := context.Background()

	if tc.BaseUrl == "" {
		return nil, fmt.Errorf("Invalid connector config. Missing URL")
	}

	// log.Debugf("Connecting to MCP server at %s", tc.Connector.BaseUrl)
	client := mcpcli.NewMcpClient(&api.ConnectorConfig{
		Provider: tc.Provider,
		BaseUrl:  tc.BaseUrl,
		ApiKey:   tc.ApiKey,
	})
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
			Kit:         kit,
			Type:        api.ToolTypeMcp,
			Name:        v.Name,
			Description: v.Description,
			Parameters:  params,
			Body:        nil,
			//
			Provider: tc.Provider,
			BaseUrl:  tc.BaseUrl,
			ApiKey:   tc.ApiKey,
			//
			// Config: tc,
		}
		meta := v.GetMeta()
		if len(meta) > 0 {
			tool.Extra = make(map[string]any)
			maps.Copy(tool.Extra, meta)
		}
		tools = append(tools, tool)
	}

	return tools, nil
}
