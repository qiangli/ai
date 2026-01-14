package conf

import (
	// "fmt"
	// "os"
	"path"
	"strings"
	"time"

	"dario.cat/mergo"
	"github.com/hashicorp/golang-lru/v2/expirable"
	"gopkg.in/yaml.v3"

	"github.com/qiangli/ai/swarm/api"
)

type AgentCacheKey struct {
	// user email
	User string
	// owner meail
	Owner string
	// agent
	Pack string
	Sub  string
}

var (
	agentCache = expirable.NewLRU[AgentCacheKey, *api.Agent](10000, nil, time.Second*900)
)

func listAgentsATM(owner string, as api.ATMSupport, packs map[string]*api.AppConfig) error {
	recs, err := as.ListAgents(owner)
	if err != nil {
		return err
	}

	// not found
	if len(recs) == 0 {
		return nil
	}

	for _, v := range recs {
		ac, err := LoadAgentsData([][]byte{[]byte(v.Content)})
		if err != nil {
			// TODO show errors for list_agents tool call
			//return fmt.Errorf("error loading agent data: %s", v.Name)
			continue
		}
		if ac == nil || len(ac.Agents) == 0 {
			// return fmt.Errorf("invalid config. no agent defined: %s", v.Name)
			continue
		}

		// correct name and add to list
		// keep store loader for loading extra resources later
		ac.Pack = strings.ToLower(v.Name)
		if _, ok := packs[ac.Pack]; ok {
			continue
		}
		for _, v := range ac.Agents {
			// v.Name = NormalizePackname(ac.Name, v.Name)
			v.Store = as
		}
		packs[ac.Pack] = ac
	}
	return nil
}

func listAgentsAsset(as api.AssetFS, root string, packs map[string]*api.AppConfig) error {
	dirs, err := as.ReadDir(root)
	if err != nil {
		return err
	}

	// not found
	if len(dirs) == 0 {
		return nil
	}

	for _, v := range dirs {
		if !v.IsDir() {
			continue
		}
		pack := v.Name()
		base := path.Join(root, pack)
		// filename := path.Join(base, "agent.yaml")
		// content, err := as.ReadFile(filename)
		// if err != nil {
		// 	if os.IsNotExist(err) {
		// 		continue
		// 	}
		// 	// return fmt.Errorf("failed to read agent asset %s: %w", v.Name(), err)
		// 	continue
		// }
		var content [][]byte
		pdirs, err := as.ReadDir(base)
		if err != nil {
			continue
		}
		for _, file := range pdirs {
			if file.IsDir() {
				continue
			}
			name := file.Name()
			if strings.HasSuffix(name, ".yaml") || strings.HasSuffix(name, ".yml") {
				if v, err := as.ReadFile(path.Join(base, name)); err != nil {
					continue
				} else {
					content = append(content, v)
				}
			}
		}
		if len(content) == 0 {
			continue
		}

		ac, err := LoadAgentsData(content)
		if err != nil {
			//return fmt.Errorf("error loading agent data: %s", v.Name())
			continue
		}
		if ac == nil || len(ac.Agents) == 0 {
			continue
		}

		// correct name and add to list
		// keep store loader for loading extra resources later
		if ac.Pack == "" {
			ac.Pack = Packname(pack)
		}
		if _, ok := packs[ac.Pack]; ok {
			continue
		}
		for _, v := range ac.Agents {
			// v.Name = NormalizePackname(ac.Name, v.Name)
			v.Store = as
			v.BaseDir = base
		}
		packs[ac.Pack] = ac
	}
	return nil
}

// LoadAgentsData loads the agent configuration from the provided YAML data.
func LoadAgentsData(data [][]byte) (*api.AppConfig, error) {
	merged := &api.AppConfig{}

	for _, v := range data {
		cfg := &api.AppConfig{}
		if err := yaml.Unmarshal(v, cfg); err != nil {
			return nil, err
		}

		if err := mergo.Merge(merged, cfg, mergo.WithAppendSlice); err != nil {
			return nil, err
		}
	}

	return merged, nil
}
