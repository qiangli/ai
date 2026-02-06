package conf

import (
	"fmt"
	"os"
	"path"
	"strings"

	// "time"

	"dario.cat/mergo"
	// "github.com/hashicorp/golang-lru/v2/expirable"
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

// var (
// 	agentCache = expirable.NewLRU[AgentCacheKey, *api.Agent](10000, nil, time.Second*900)
// )

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

		ac.Store = as
		ac.BaseDir = ""
		packs[ac.Pack] = ac
	}
	return nil
}

func listAgentsAsset(as api.AssetFS, root string, packs map[string]*api.AppConfig) error {
	files, err := as.ReadDir(root)
	if err != nil {
		return err
	}

	// not found
	if len(files) == 0 {
		return nil
	}

	for _, file := range files {
		var content [][]byte
		var base string
		var pack string
		if !file.IsDir() {
			name := file.Name()
			data, err := as.ReadFile(path.Join(root, name))
			if err != nil {
				if os.IsNotExist(err) {
					continue
				}
				return fmt.Errorf("failed to read agent asset %s: %w", name, err)
			}
			content = [][]byte{data}
			base = root
			pack = strings.TrimSuffix(name, path.Ext(name))
		} else {
			pack = file.Name()
			base := path.Join(root, pack)

			pdirs, err := as.ReadDir(base)
			if err != nil {
				continue
			}
			for _, v := range pdirs {
				if v.IsDir() {
					continue
				}
				name := v.Name()
				if strings.HasSuffix(name, ".yaml") || strings.HasSuffix(name, ".yml") {
					if data, err := as.ReadFile(path.Join(base, name)); err != nil {
						continue
					} else {
						content = append(content, data)
					}
				}
			}
		}
		if len(content) == 0 {
			continue
		}

		//
		ac, err := LoadAgentsData(content)
		if err != nil {
			return fmt.Errorf("error loading agent data: %s", file.Name())
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

		ac.Store = as
		ac.BaseDir = base
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
