package conf

import (
	"fmt"
	"path"

	"github.com/qiangli/ai/swarm/api"
)

type assetManager struct {
	user    *api.User
	secrets api.SecretStore
	assets  []api.AssetStore
}

func NewAssetManager(user *api.User) api.AssetManager {
	return &assetManager{
		user:   user,
		assets: make([]api.AssetStore, 0),
	}
}

func (r *assetManager) AddStore(store api.AssetStore) {
	r.assets = append(r.assets, store)
}

func (r *assetManager) SearchAgent(owner string, pack string) (*api.Record, error) {
	for _, v := range r.assets {
		// try search first
		if as, ok := v.(api.ATMSupport); ok {
			if a, err := as.SearchAgent(r.user.Email, owner, pack); err == nil && a != nil {
				a.Store = as
				return a, nil
			}
		} else if as, ok := v.(api.AssetFS); ok {
			if _, err := as.ReadFile(path.Join("agents", pack, "agent.yaml")); err == nil {
				return &api.Record{
					Owner: owner,
					Name:  pack,
					Store: as,
				}, nil
			}
		}
	}

	// TODO  support searching for sub agent?
	return nil, nil
}

func (r *assetManager) ListAgent(owner string) (map[string]*api.AgentsConfig, error) {
	var packs = make(map[string]*api.AgentsConfig)
	for _, v := range r.assets {
		if as, ok := v.(api.ATMSupport); ok {
			if err := loadAgentsATM(owner, as, packs); err != nil {
				return nil, err
			}
		} else if as, ok := v.(api.AssetFS); ok {
			if err := loadAgentsAsset(as, "agents", packs); err != nil {
				return nil, err
			}
		}
	}

	var agents = make(map[string]*api.AgentsConfig)
	// add sub agent to the map as well
	for _, v := range packs {
		if len(v.Agents) == 0 {
			continue
		}
		// Register the agent configurations
		for _, agent := range v.Agents {
			if _, exists := agents[agent.Name]; exists {
				continue
			}
			// Register the agents configuration
			agents[agent.Name] = v

			if v.MaxTurns == 0 {
				v.MaxTurns = defaultMaxTurns
			}
			if v.MaxTime == 0 {
				v.MaxTime = defaultMaxTime
			}
		}
	}

	if len(agents) == 0 {
		return nil, fmt.Errorf("no agent configurations found")
	}
	return agents, nil
}

func (r *assetManager) FindToolkit(owner string, kit string) (*api.ToolsConfig, error) {
	var content []byte
	for _, v := range r.assets {
		if as, ok := v.(api.ATMSupport); ok {
			if v, err := as.RetrieveTool(owner, kit); err != nil {
				// return nil, err
				continue
			} else {
				content = []byte(v.Content)
				break
			}
		} else if as, ok := v.(api.AssetFS); ok {
			if v, err := as.ReadFile(path.Join("tools", kit+".yaml")); err != nil {
				// return nil, err
				continue
			} else {
				content = v
				break
			}
		}
	}

	if len(content) == 0 {
		return nil, nil
	}

	tc, err := loadToolData([][]byte{content})
	if err != nil {
		return nil, err
	}
	// NOTE: this may change
	if tc == nil || (len(tc.Tools) == 0 && tc.Connector == nil) {
		return nil, fmt.Errorf("invalid config. no tools defined: %s", kit)
	}

	//
	tc.Kit = kit

	return tc, nil
}

func (r *assetManager) FindModels(owner string, alias string) (*api.ModelsConfig, error) {
	var content []byte
	for _, v := range r.assets {
		if as, ok := v.(api.ATMSupport); ok {
			if v, err := as.RetrieveModel(owner, alias); err != nil {
				// return nil, err
				continue
			} else {
				content = []byte(v.Content)
				break
			}
		} else if as, ok := v.(api.AssetFS); ok {
			if v, err := as.ReadFile(path.Join("models", alias+".yaml")); err != nil {
				// return nil, err
				continue
			} else {
				content = v
				break
			}
		}
	}

	if len(content) == 0 {
		return nil, nil
	}

	mc, err := loadModelsData([][]byte{content})
	if err != nil {
		return nil, fmt.Errorf("failed to load models: %s. %v", alias, err)
	}
	if mc == nil || len(mc.Models) == 0 {
		return nil, fmt.Errorf("invalid models config: %s", alias)
	}

	//
	mc.Alias = alias

	return mc, nil
}
