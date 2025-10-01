package conf

import (
	// "context"
	// "embed"
	"fmt"
	// "path/filepath"
	"os"

	"github.com/qiangli/ai/swarm/api"
	// "github.com/qiangli/ai/swarm/resource"
)

type assetManager struct {
	user   *api.User
	assets map[string]api.AssetStore
}

func NewAssetManager(user *api.User) api.AssetManager {
	return &assetManager{
		user:   user,
		assets: make(map[string]api.AssetStore),
	}
}

func (r *assetManager) GetStore(key string) (api.AssetStore, error) {
	if v, ok := r.assets[key]; ok {
		return v, nil
	}
	return nil, os.ErrNotExist
}

func (r *assetManager) AddStore(key string, store api.AssetStore) {
	r.assets[key] = store
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
		// log.GetLogger(ctx).Debugf("Registering agent: %s with %d configurations\n", name, len(v.Agents))
		if len(v.Agents) == 0 {
			// log.GetLogger(ctx).Debugf("No agents found in config: %s\n", name)
			continue
		}
		// Register the agent configurations
		for _, agent := range v.Agents {
			if _, exists := agents[agent.Name]; exists {
				// log.GetLogger(ctx).Debugf("Duplicate agent name found: %s, skipping registration\n", agent.Name)
				continue
			}
			// Register the agents configuration
			agents[agent.Name] = v
			// log.GetLogger(ctx).Debugf("Registered agent: %s\n", agent.Name)

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
	// log.GetLogger(ctx).Debugf("Initialized %d agent configurations\n", len(agents))
	return agents, nil
}

func (r *assetManager) FindToolkit(owner string, kit string) (*api.ToolsConfig, error) {
	return nil, nil
}

func (r *assetManager) FindModels(owner string, alias string) (*api.ModelsConfig, error) {
	return nil, nil
}

// func (r *assetManager) loadAgentsConfig() (map[string]*api.AgentsConfig, error) {

// }

// func (r *assetManager) loadAgentsConfig(ctx context.Context, app *api.AppConfig) (map[string]*api.AgentsConfig, error) {
// 	var packs = make(map[string]*api.AgentsConfig)

// 	for _, rs := range r.assets {
// 		LoadAgentsAsset(rs, "agents", packs)
// 	}

// 	// // web
// 	// if app.AgentResource != nil && len(app.AgentResource.Resources) > 0 {
// 	// 	if err := LoadWebAgentsConfig(ctx, app.AgentResource.Resources, packs); err != nil {
// 	// 		// log.GetLogger(ctx).Errorf("Failed load agents from web resources: %v\n", err)
// 	// 	}
// 	// }

// 	// // external/custom
// 	// if err := LoadFileAgentsConfig(ctx, app.Base, packs); err != nil {
// 	// 	// log.GetLogger(ctx).Errorf("Failed to load custom agents: %v\n", err)
// 	// }

// 	// // default
// 	// if err := LoadResourceAgentsConfig(ctx, resource.ResourceFS, packs); err != nil {
// 	// 	return nil, err
// 	// }

// 	return packs, nil
// }

// func LoadResourceAgentsConfig(ctx context.Context, fs embed.FS, packs map[string]*api.AgentsConfig) error {
// 	rs := &resource.ResourceStore{
// 		FS:   fs,
// 		Base: "standard",
// 	}
// 	return LoadAgentsAsset(ctx, rs, "agents", packs)
// }

// func LoadFileAgentsConfig(ctx context.Context, base string, packs map[string]*api.AgentsConfig) error {
// 	abs, err := filepath.Abs(base)
// 	if err != nil {
// 		return fmt.Errorf("failed to get absolute path for %s: %w", base, err)
// 	}
// 	// check if abs exists
// 	if _, err := os.Stat(abs); os.IsNotExist(err) {
// 		// log.GetLogger(ctx).Debugf("Path does not exist: %s\n", abs)
// 		return nil
// 	}

// 	fs := &resource.FileStore{
// 		Base: abs,
// 	}
// 	return LoadAgentsAsset(ctx, fs, "agents", packs)
// }

// func LoadWebAgentsConfig(ctx context.Context, resources []*api.Resource, packs map[string]*api.AgentsConfig) error {
// 	for _, v := range resources {
// 		ws := &resource.WebStore{
// 			Base:  v.Base,
// 			Token: v.Token,
// 		}
// 		if err := LoadAgentsAsset(ctx, ws, "agents", packs); err != nil {
// 			// log.GetLogger(ctx).Errorf("Failed to load config. base: %s error: %v\n", v.Base, err)
// 		}
// 	}
// 	return nil
// }
