package conf

import (
	"fmt"
	"path"
	"strings"
	"time"

	"dario.cat/mergo"
	"github.com/hashicorp/golang-lru/v2/expirable"
	// log "github.com/sirupsen/logrus"
	"gopkg.in/yaml.v3"

	// "github.com/qiangli/ai/swarm/agent/internal/db"
	"github.com/qiangli/ai/swarm/api"
)

type ToolkitCacheKey struct {
	Owner string
	Kit   string
}

var (
	toolkitCache = expirable.NewLRU[ToolkitCacheKey, []*api.ToolFunc](10000, nil, time.Second*180)
)

func loadToolFunc(owner, s string) ([]*api.ToolFunc, error) {
	// kit__name
	// kit:*
	// kit:name
	var kit, name string

	if strings.Index(s, "__") > 0 {
		// call time - the name should never be empty
		kit, name = split2(s, "__", "")
		if name == "" {
			return nil, fmt.Errorf("invalid tool call id: %s", s)
		}
	} else {
		// load time
		kit, name = split2(s, ":", "*")
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

	if tc, err := retrieveActiveToolkit(owner, kit); err != nil {
		return nil, err
	} else {
		if tc != nil {
			v, err := loadTools(tc, owner)
			if err != nil {
				return nil, err
			}
			toolkitCache.Add(key, v)
			return filter(v)
		}
	}

	// default
	if tc, ok := standardTools[kit]; ok {
		v, err := loadTools(tc, owner)
		if err != nil {
			return nil, err
		}
		return filter(v)
	}

	return nil, fmt.Errorf("tool not found: %s", s)
}

func loadToolsAsset(as api.AssetStore, base string, kits map[string]*api.ToolsConfig) error {
	dirs, err := as.ReadDir(base)
	if err != nil {
		return fmt.Errorf("failed to read directory: %v", err)
	}
	for _, dir := range dirs {
		if dir.IsDir() {
			continue
		}
		f, err := as.ReadFile(path.Join(base, dir.Name()))
		if err != nil {
			return fmt.Errorf("failed to read tool file %s: %w", dir.Name(), err)
		}
		if len(f) == 0 {
			continue
		}
		kit, err := loadToolData([][]byte{f})
		if err != nil {
			return err
		}
		kits[kit.Kit] = kit
	}

	return nil
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

// return nil if not found
func retrieveActiveToolkit(
	owner string,
	kitName string,
) (*api.ToolsConfig, error) {
	// t, found, err := db.GetActiveToolByEmail(owner, kitName)
	// if err != nil {
	// 	return nil, err
	// }

	// // not found
	// if !found {
	// 	return nil, nil
	// }

	var content []byte
	tc, err := loadToolData([][]byte{content})
	if err != nil {
		return nil, err
	}
	// NOTE: this may change
	if tc == nil || (len(tc.Tools) == 0 && tc.Connector == nil) {
		return nil, fmt.Errorf("invalid config. no tools defined: %s", kitName)
	}

	//
	tc.Kit = kitName

	return tc, nil
}

func loadTools(tc *api.ToolsConfig, owner string) ([]*api.ToolFunc, error) {
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
				Provider: nvl(v.Provider, tc.Provider),
				BaseUrl:  nvl(v.BaseUrl, tc.BaseUrl),
				ApiKey:   nvl(v.ApiKey, tc.ApiKey),
				//
				Config: tc,
			}
			if tool.Type == "" {
				return nil, fmt.Errorf("Missing tool type: %s %s", tc.Kit, tool.Name)
			}

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
