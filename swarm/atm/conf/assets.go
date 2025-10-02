package conf

import (
	"fmt"

	"github.com/qiangli/ai/swarm/api"
)

type assetManager struct {
	user   *api.User
	assets []api.AssetStore
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

func (r *assetManager) SearchAgent(owner string, pack string) (*api.AgentsConfig, error) {
	return nil, nil
}

func (r *assetManager) ListAgent(owner string) (map[string]*api.AgentsConfig, error) {
	var agents = make(map[string]*api.AgentsConfig)

	var packs = make(map[string]*api.AgentsConfig)

	for _, rs := range r.assets {
		LoadAgentsAsset(rs, "agents", packs)
	}

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
	return nil, nil
}

func (r *assetManager) FindModels(owner string, alias string) (*api.ModelsConfig, error) {
	return nil, nil
}
