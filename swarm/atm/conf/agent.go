package conf

import (
	"fmt"
	"os"
	"path"
	"strings"
	"time"

	"dario.cat/mergo"
	"github.com/hashicorp/golang-lru/v2/expirable"
	"gopkg.in/yaml.v3"

	"github.com/qiangli/ai/swarm/api"
	"github.com/qiangli/ai/swarm/util"
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

func normalizePackname(pack, name string) string {
	ensure := func() string {
		// pack name
		if name == "" {
			return pack
		}
		parts := strings.SplitN(name, "/", 2)
		if len(parts) == 1 {
			return pack
		}
		return pack + "/" + parts[1]
	}
	return util.NormalizedName(ensure())
}

func listAgentsATM(owner string, as api.ATMSupport, packs map[string]*api.AgentsConfig) error {
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
			return err
		}
		if ac == nil || len(ac.Agents) == 0 {
			return fmt.Errorf("invalid config. no agent defined: %s", v.Name)
		}

		// correct name and add to list
		// keep store loader for loading extra resources later
		ac.Name = strings.ToLower(v.Name)
		if _, ok := packs[ac.Name]; ok {
			continue
		}
		for _, v := range ac.Agents {
			v.Name = normalizePackname(ac.Name, v.Name)
			v.Store = as
		}
		packs[ac.Name] = ac
	}
	return nil
}

func listAgentsAsset(as api.AssetFS, root string, packs map[string]*api.AgentsConfig) error {
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
		base := path.Join(root, v.Name())
		filename := path.Join(base, "agent.yaml")
		content, err := as.ReadFile(filename)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return fmt.Errorf("failed to read agent asset %s: %w", v.Name(), err)
		}
		if len(content) == 0 {
			continue
		}

		ac, err := LoadAgentsData([][]byte{content})
		if err != nil {
			return err
		}
		if ac == nil || len(ac.Agents) == 0 {
			// return fmt.Errorf("invalid config. no agent defined: %s", v.Name())
			continue
		}

		// correct name and add to list
		// keep store loader for loading extra resources later
		if ac.Name == "" {
			ac.Name = Packname(v.Name())
		}
		if _, ok := packs[ac.Name]; ok {
			continue
		}
		for _, v := range ac.Agents {
			v.Name = normalizePackname(ac.Name, v.Name)
			v.Store = as
			v.BaseDir = base
		}
		packs[ac.Name] = ac
	}
	return nil
}

// LoadAgentsData loads the agent configuration from the provided YAML data.
func LoadAgentsData(data [][]byte) (*api.AgentsConfig, error) {
	merged := &api.AgentsConfig{}

	for _, v := range data {
		cfg := &api.AgentsConfig{}
		if err := yaml.Unmarshal(v, cfg); err != nil {
			return nil, err
		}

		if err := mergo.Merge(merged, cfg, mergo.WithAppendSlice); err != nil {
			return nil, err
		}
	}

	return merged, nil
}
