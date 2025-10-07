package conf

import (
	"fmt"
	"maps"
	"os"
	"path"
	"time"

	"dario.cat/mergo"
	"github.com/hashicorp/golang-lru/v2/expirable"
	"gopkg.in/yaml.v3"

	"github.com/qiangli/ai/swarm/api"
)

type ToolkitCacheKey struct {
	Owner string
	Kit   string
}

var (
	toolkitCache = expirable.NewLRU[ToolkitCacheKey, []*api.ToolFunc](10000, nil, time.Second*900)
)

func LoadToolFunc(owner, s string, secrets api.SecretStore, assets api.AssetManager) ([]*api.ToolFunc, error) {
	kit, name := api.KitName(s).Decode()
	if name == "" {
		return nil, fmt.Errorf("invalid tool call id: %s", s)
	}

	// return tool or the kit
	filter := func(tools []*api.ToolFunc) ([]*api.ToolFunc, error) {
		var filtered = filterTool(tools, kit, name)
		if len(filtered) == 0 {
			return nil, fmt.Errorf("no such tool: %s", s)
		}
		return filtered, nil
	}

	key := ToolkitCacheKey{owner, kit}

	if v, ok := toolkitCache.Get(key); ok {
		return filter(v)
	}

	// agent:name
	if kit == api.ToolTypeAgent {
		ac, err := assets.FindAgent(owner, name)
		if err != nil {
			return nil, err
		}
		return loadAgentTool(ac, name)
	}

	tc, err := assets.FindToolkit(owner, kit)
	if err != nil {
		return nil, err
	}

	if tc != nil {
		v, err := loadTools(tc, owner, secrets)
		if err != nil {
			return nil, err
		}
		toolkitCache.Add(key, v)
		return filter(v)
	}

	return nil, fmt.Errorf("tool func not found: %s", s)
}

func loadToolData(data [][]byte) (*api.ToolsConfig, error) {
	merged := &api.ToolsConfig{}

	for _, v := range data {
		tc := &api.ToolsConfig{}
		if err := yaml.Unmarshal(v, tc); err != nil {
			return nil, err
		}

		if err := mergo.Merge(merged, tc, mergo.WithAppendSlice); err != nil {
			return nil, err
		}
	}

	// fill defaults
	for _, v := range merged.Tools {
		if v.Type == "" {
			v.Type = merged.Type
		}

		if v.Provider == "" {
			v.Provider = merged.Provider
		}
		if v.BaseUrl == "" {
			v.BaseUrl = merged.BaseUrl
		}
		if v.ApiKey == "" {
			v.ApiKey = merged.ApiKey
		}
	}

	if v := merged.Connector; v != nil {
		if v.BaseUrl == "" {
			v.BaseUrl = merged.BaseUrl
		}
		if v.Provider == "" {
			v.Provider = merged.Provider
		}
		if v.ApiKey == "" {
			v.ApiKey = merged.ApiKey
		}
	}

	return merged, nil
}

func loadTools(tc *api.ToolsConfig, owner string, secrets api.SecretStore) ([]*api.ToolFunc, error) {
	var toolMap = make(map[string]*api.ToolFunc)

	conditionMet := func(name string, c *api.ToolCondition) bool {
		return true
	}

	// for _, tc := range kits {
	if len(tc.Tools) > 0 {
		for _, v := range tc.Tools {
			// log.Debugf("Kit: %s tool: %s - %s\n", tc.Kit, v.Name, v.Description)

			// condition check
			if !conditionMet(v.Name, v.Condition) {
				continue
			}

			tool := &api.ToolFunc{
				Kit:         tc.Kit,
				Type:        v.Type,
				Name:        v.Name,
				Description: v.Description,
				Parameters:  v.Parameters,
				Body:        v.Body,
				//
				Agent: v.Agent,
				//
				Provider: nvl(v.Provider, tc.Provider),
				BaseUrl:  nvl(v.BaseUrl, tc.BaseUrl),
				ApiKey:   nvl(v.ApiKey, tc.ApiKey),
				//
				Config: tc,
			}
			if tool.Type == "" {
				return nil, fmt.Errorf("Missing tool type: %s %s", tc.Kit, tool.Name)
			}

			// TODO merge description/parameters for agent tool?

			toolMap[tool.ID()] = tool
		}
	}

	// connector mcp
	if tc.Connector != nil {
		var token string
		apiKey := nvl(tc.Connector.ApiKey, tc.ApiKey)
		if v, err := secrets.Get(owner, apiKey); err != nil {
			return nil, err
		} else {
			token = v
		}
		mcpTools, err := listMcpTools(tc, token)
		if err != nil {
			return nil, err
		}
		for _, tool := range mcpTools {
			toolMap[tool.ID()] = tool
		}
	}

	var tools = make([]*api.ToolFunc, 0)
	for _, v := range toolMap {
		tools = append(tools, v)
	}
	return tools, nil
}

func loadAgentTool(ac *api.AgentsConfig, name string) ([]*api.ToolFunc, error) {
	for _, c := range ac.Agents {
		if c.Name == name {
			var params = map[string]any{
				"type": "object",
				"properties": map[string]any{
					"query": map[string]any{
						"type":        "string",
						"description": "The user input",
					},
				},
			}
			maps.Copy(params, c.Parameters)

			tool := &api.ToolFunc{
				Kit:         "agent",
				Type:        api.ToolTypeAgent,
				Name:        c.Name,
				Description: c.Description,
				Parameters:  params,
				Body:        nil,
				//
				Agent: c.Name,
				//
				Config: &api.ToolsConfig{
					Kit:  "agent",
					Type: api.ToolTypeAgent,
				},
			}
			return []*api.ToolFunc{tool}, nil
		}
	}

	return nil, nil
}

func listToolkitATM(owner string, as api.ATMSupport, kits map[string]*api.ToolsConfig) error {
	recs, err := as.ListTools(owner)
	if err != nil {
		return err
	}

	// not found
	if len(recs) == 0 {
		return nil
	}

	for _, v := range recs {
		tc, err := loadToolData([][]byte{[]byte(v.Content)})
		if err != nil {
			return err
		}
		if tc == nil || len(tc.Tools) == 0 {
			return fmt.Errorf("invalid config. no tool defined: %s", v.Name)
		}
		//
		if tc.Kit == "" {
			tc.Kit = kitName(v.Name)
		}
		if _, ok := kits[tc.Kit]; ok {
			continue
		}

		kits[tc.Kit] = tc
	}
	return nil
}

func listToolkitAsset(as api.AssetFS, base string, kits map[string]*api.ToolsConfig) error {
	dirs, err := as.ReadDir(base)
	if err != nil {
		return err
	}

	// not found
	if len(dirs) == 0 {
		return nil
	}

	for _, v := range dirs {
		if v.IsDir() {
			continue
		}
		content, err := as.ReadFile(path.Join(base, v.Name()))
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return fmt.Errorf("failed to read tool asset %s: %w", v.Name(), err)
		}
		if len(content) == 0 {
			continue
		}

		tc, err := loadToolData([][]byte{content})
		if err != nil {
			return err
		}
		if tc == nil || len(tc.Tools) == 0 {
			// TODO mcp
			continue
		}

		//
		if tc.Kit == "" {
			tc.Kit = kitName(v.Name())
		}
		if _, ok := kits[tc.Kit]; ok {
			continue
		}
		kits[tc.Kit] = tc
	}
	return nil
}
