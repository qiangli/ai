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
	kit, name := api.Kitname(s).Decode()
	if name == "" {
		return nil, fmt.Errorf("invalid tool call id: %s", s)
	}

	// return tool or the kit
	filter := func(tools []*api.ToolFunc) ([]*api.ToolFunc, error) {
		var filtered = filterTool(tools, kit, name)
		if len(filtered) == 0 {
			return nil, fmt.Errorf("no such tool: %q", s)
		}
		return filtered, nil
	}

	key := ToolkitCacheKey{owner, kit}

	if v, ok := toolkitCache.Get(key); ok {
		return filter(v)
	}

	// builtin "agent:" toolkit
	// any agent can be used a tool
	// @name
	// agent:name
	if kit == string(api.ToolTypeAgent) {
		pack, sub := api.Packname(name).Decode()
		ac, err := assets.FindAgent(owner, pack)
		if err != nil {
			return nil, err
		}
		v, err := LoadAgentTool(ac, sub)
		if err != nil {
			return nil, err
		}
		if v == nil {
			return nil, fmt.Errorf("agent tool not found: %s", name)
		}
		return []*api.ToolFunc{v}, nil
	}

	//
	tc, err := assets.FindToolkit(owner, kit)
	if err != nil {
		return nil, err
	}

	if tc != nil {
		v, err := LoadTools(tc, owner, secrets)
		if err != nil {
			return nil, err
		}
		toolkitCache.Add(key, v)
		return filter(v)
	}

	return nil, fmt.Errorf("tool func not found: %s", s)
}

// try load local scope (inline) in the same config as the referenced agent
func LoadLocalToolFunc(local *api.AppConfig, owner, s string, secrets api.SecretStore, assets api.AssetManager) ([]*api.ToolFunc, error) {
	kit, name := api.Kitname(s).Decode()
	if name == "" {
		return nil, fmt.Errorf("invalid tool call id: %s", s)
	}

	// return tool or the kit
	filter := func(tools []*api.ToolFunc) ([]*api.ToolFunc, error) {
		var filtered = filterTool(tools, kit, name)
		if len(filtered) == 0 {
			return nil, fmt.Errorf("no such tool: %q", s)
		}
		return filtered, nil
	}

	// builtin "agent:" toolkit
	// any agent can be used as a tool
	// agent:<agent name>
	if kit == string(api.ToolTypeAgent) {
		if name == local.Name {
			return nil, fmt.Errorf("recursion not supported: %s", name)
		}
		for _, v := range local.Agents {
			if name == v.Name {
				v, err := LoadAgentTool(local, name)
				if err != nil {
					return nil, err
				}
				return []*api.ToolFunc{v}, nil
			}
		}
	}

	// if kit refers to a kit in the same local config as the agents.
	// local takes precedence.
	var tc *api.AppConfig
	if kit == local.Kit {
		tc = &api.AppConfig{
			Kit:   kit,
			Type:  local.Type,
			Tools: local.Tools,
			//
			Provider: local.Provider,
			BaseUrl:  local.BaseUrl,
			ApiKey:   local.ApiKey,
		}
	}

	if tc != nil {
		v, err := LoadTools(tc, owner, secrets)
		if err != nil {
			return nil, err
		}
		return filter(v)
	}

	// no error, external kit will be resolved next
	return nil, nil
}

func LoadToolData(data [][]byte) (*api.AppConfig, error) {
	merged := &api.AppConfig{}

	for _, v := range data {
		tc := &api.AppConfig{}
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

	// if v := merged.Connector; v != nil {
	// 	if v.BaseUrl == "" {
	// 		v.BaseUrl = merged.BaseUrl
	// 	}
	// 	if v.Provider == "" {
	// 		v.Provider = merged.Provider
	// 	}
	// 	if v.ApiKey == "" {
	// 		v.ApiKey = merged.ApiKey
	// 	}
	// }

	return merged, nil
}

func LoadTools(tc *api.AppConfig, owner string, secrets api.SecretStore) ([]*api.ToolFunc, error) {
	var toolMap = make(map[string]*api.ToolFunc)

	// conditionMet := func(name string, c *api.ToolCondition) bool {
	// 	return true
	// }

	for _, v := range tc.Tools {
		// log.Debugf("Kit: %s tool: %s - %s\n", tc.Kit, v.Name, v.Description)

		// // condition check
		// if !conditionMet(v.Name, v.Condition) {
		// 	continue
		// }

		var toolType = nvl(v.Type, tc.Type)
		if toolType == "" {
			return nil, fmt.Errorf("Missing tool type. kit: %s tool: %s", tc.Kit, v.Name)
		}

		// load separately
		if toolType == string(api.ToolTypeMcp) {
			continue
		}

		tool := &api.ToolFunc{
			Kit:         tc.Kit,
			Type:        api.ToolType(toolType),
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
			// Config: tc,
		}
		// TODO merge description/parameters for agent tool?
		toolMap[tool.ID()] = tool
	}

	// contact mcp servers and fetch the list of tools
	for _, v := range tc.Tools {
		var toolType = nvl(v.Type, tc.Type)
		if toolType != string(api.ToolTypeMcp) {
			continue
		}
		var token string
		apiKey := nvl(v.ApiKey, tc.ApiKey)
		if v, err := secrets.Get(owner, apiKey); err != nil {
			return nil, err
		} else {
			token = v
		}
		connector := &api.ConnectorConfig{
			Provider: nvl(v.Provider, tc.Provider),
			BaseUrl:  nvl(v.BaseUrl, tc.BaseUrl),
			ApiKey:   nvl(v.ApiKey, tc.ApiKey),
		}
		mcpTools, err := listMcpTools(tc.Kit, connector, token)
		if err != nil {
			return nil, err
		}

		// return true if extra is empty or has the name/val in the filter
		met := func(extra map[string]any) bool {
			if len(extra) == 0 || len(v.Filter) == 0 {
				return true
			}
			for key, f := range v.Filter {
				if m, ok := extra[key]; ok && m == f {
					return true
				}
			}
			return false
		}
		for _, tool := range mcpTools {
			if met(tool.Extra) {
				toolMap[tool.ID()] = tool
			}
		}
	}

	// TODO deprecated in favor of tools
	// connector mcp
	// if tc.Connector != nil {
	// 	var token string
	// 	apiKey := nvl(tc.Connector.ApiKey, tc.ApiKey)
	// 	if v, err := secrets.Get(owner, apiKey); err != nil {
	// 		return nil, err
	// 	} else {
	// 		token = v
	// 	}
	// 	mcpTools, err := listMcpTools(tc, token)
	// 	if err != nil {
	// 		return nil, err
	// 	}
	// 	for _, tool := range mcpTools {
	// 		toolMap[tool.ID()] = tool
	// 	}
	// }

	var tools = make([]*api.ToolFunc, 0)
	for _, v := range toolMap {
		tools = append(tools, v)
	}
	return tools, nil
}

func LoadAgentTool(ac *api.AppConfig, sub string) (*api.ToolFunc, error) {
	if ac == nil {
		return nil, fmt.Errorf("nil config: %s", sub)
	}
	for _, c := range ac.Agents {
		if c.Name == sub {
			var params = map[string]any{}
			// "type": "object",
			// "properties": map[string]any{
			// 	"query": map[string]any{
			// 		"type":        "string",
			// 		"description": "The user input",
			// 	},
			// },
			// }
			maps.Copy(params, c.Parameters)
			// required for agent tool.
			if _, ok := params["type"]; !ok {
				params["type"] = "object"
			}
			if v, ok := params["properties"]; !ok {
				params["properties"] = map[string]any{
					"query": map[string]any{
						"type":        "string",
						"description": "The user input",
					},
				}
			} else {
				m, ok := v.(map[string]any)
				if !ok {
					return nil, fmt.Errorf("invalid parameters %v", v)
				}
				if _, ok := m["query"]; !ok {
					m["query"] = map[string]any{
						"type":        "string",
						"description": "The user input",
					}
				}
			}

			pn := api.NewPackname(ac.Pack, sub)

			tool := &api.ToolFunc{
				Kit:  string(api.ToolTypeAgent),
				Type: api.ToolTypeAgent,
				// func name: pack/sub?
				// Name:        c.Name,
				Name:        pn.String(),
				Description: c.Description,
				Parameters:  params,
				Body:        nil,
				//
				Agent: pn.String(),
				// TODO
				// for flow_type/actions
				// agent will be recreated with full args
				Arguments: c.Arguments,
			}

			return tool, nil
		}
	}

	return nil, nil
}

func listToolkitATM(owner string, as api.ATMSupport, kits map[string]*api.AppConfig) error {
	recs, err := as.ListTools(owner)
	if err != nil {
		return err
	}

	// not found
	if len(recs) == 0 {
		return nil
	}

	for _, v := range recs {
		tc, err := LoadToolData([][]byte{[]byte(v.Content)})
		if err != nil {
			return err
		}
		if tc == nil || len(tc.Tools) == 0 {
			return fmt.Errorf("invalid config. no tool defined: %s", v.Name)
		}
		//
		if tc.Kit == "" {
			tc.Kit = Kitname(v.Name)
		}
		if _, ok := kits[tc.Kit]; ok {
			continue
		}

		kits[tc.Kit] = tc
	}
	return nil
}

func listToolkitAsset(as api.AssetFS, base string, kits map[string]*api.AppConfig) error {
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

		tc, err := LoadToolData([][]byte{content})
		if err != nil {
			return fmt.Errorf("error loading tool data: %s\n", v.Name())
		}
		if tc == nil || len(tc.Tools) == 0 {
			// TODO mcp
			continue
		}

		//
		if tc.Kit == "" {
			tc.Kit = Kitname(v.Name())
		}
		if _, ok := kits[tc.Kit]; ok {
			continue
		}
		kits[tc.Kit] = tc
	}
	return nil
}
