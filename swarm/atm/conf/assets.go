package conf

import (
	"bytes"
	"fmt"
	"path"
	"strings"

	"github.com/qiangli/ai/swarm/api"
)

type assetManager struct {
	secrets api.SecretStore
	assets  []api.AssetStore
}

func NewAssetManager() api.AssetManager {
	return &assetManager{
		assets: make([]api.AssetStore, 0),
	}
}

func (r *assetManager) AddStore(store api.AssetStore) {
	r.assets = append(r.assets, store)
}

// func (r *assetManager) SearchAgent(owner string, pack string) (*api.Record, error) {
// 	for _, v := range r.assets {
// 		// try search first
// 		if as, ok := v.(api.ATMSupport); ok {
// 			if a, err := as.SearchAgent(owner, pack); err == nil && a != nil {
// 				a.Store = as
// 				return a, nil
// 			}
// 		} else if as, ok := v.(api.AssetFS); ok {
// 			if _, err := as.ReadFile(path.Join("agents", pack, "agent.yaml")); err == nil {
// 				return &api.Record{
// 					Owner: owner,
// 					Name:  pack,
// 					Store: as,
// 				}, nil
// 			}
// 		}
// 	}

// 	// TODO  support searching for sub agent?
// 	return nil, nil
// }

func (r *assetManager) ListAgent(owner string) (map[string]*api.AppConfig, error) {
	var packs = make(map[string]*api.AppConfig)
	for _, v := range r.assets {
		if as, ok := v.(api.ATMSupport); ok {
			if err := listAgentsATM(owner, as, packs); err != nil {
				// return nil, err
				continue
			}
		} else if as, ok := v.(api.AssetFS); ok {
			if err := listAgentsAsset(as, "agents", packs); err != nil {
				// return nil, err
				continue
			}
		}
	}

	var agents = make(map[string]*api.AppConfig)
	// add sub agent to the map as well
	for _, v := range packs {
		if len(v.Agents) == 0 {
			continue
		}
		// Register the sub agent
		for _, sub := range v.Agents {
			if _, exists := agents[sub.Name]; exists {
				continue
			}
			agents[sub.Name] = v
		}
	}

	if len(agents) == 0 {
		return nil, fmt.Errorf("no agent configurations found")
	}
	return agents, nil
}

func (r *assetManager) FindAgent(owner string, pack string) (*api.AppConfig, error) {
	var content [][]byte
	var asset api.AssetStore
	for _, v := range r.assets {
		if as, ok := v.(api.ATMSupport); ok {
			if v, err := as.RetrieveAgent(owner, pack); err != nil {
				continue
			} else {
				content = [][]byte{[]byte(v.Content)}
				asset = as
				break
			}
		} else if as, ok := v.(api.AssetFS); ok {
			dirs, err := as.ReadDir(path.Join("agents", pack))
			if err != nil {
				continue
			}
			for _, file := range dirs {
				if file.IsDir() {
					continue
				}
				name := file.Name()
				if strings.HasSuffix(name, ".yaml") || strings.HasSuffix(name, ".yml") {
					if v, err := as.ReadFile(path.Join("agents", pack, name)); err != nil {
						continue
					} else {
						content = append(content, v)
						asset = as
						break
					}
				}
			}
			// if v, err := as.ReadFile(path.Join("agents", pack, "agent.yaml")); err != nil {
			// 	continue
			// } else {
			// 	content = v
			// 	asset = as
			// 	break
			// }
		}
	}

	if len(content) == 0 {
		return nil, nil
	}

	ac, err := LoadAgentsData(content)
	if err != nil {
		return nil, fmt.Errorf("error loading agent data: %s", pack)
	}
	if ac == nil || len(ac.Agents) == 0 {
		return nil, fmt.Errorf("invalid config. no agent defined: %s", pack)
	}

	ac.Name = pack
	if len(content) == 1 {
		ac.RawContent = content[0]
	} else {
		var buffer bytes.Buffer
		for _, part := range content {
			buffer.Write(part)
		}
		ac.RawContent = buffer.Bytes()
	}

	// sub agents
	for _, v := range ac.Agents {
		// v.Name = NormalizePackname(pack, v.Name)
		v.Store = asset
		// TODO base for resource asset? not used and not a problem for now
	}
	return ac, nil
}

func (r *assetManager) ListToolkit(owner string) (map[string]*api.AppConfig, error) {
	var kits = make(map[string]*api.AppConfig)
	for _, v := range r.assets {
		if as, ok := v.(api.ATMSupport); ok {
			if err := listToolkitATM(owner, as, kits); err != nil {
				// return nil, err
				continue
			}
		} else if as, ok := v.(api.AssetFS); ok {
			if err := listToolkitAsset(as, "tools", kits); err != nil {
				// return nil, err
				continue
			}
		}
	}

	if len(kits) == 0 {
		return nil, fmt.Errorf("no tool configurations found")
	}
	return kits, nil
}

func (r *assetManager) FindToolkit(owner string, kit string) (*api.AppConfig, error) {
	var content []byte
	for _, v := range r.assets {
		if as, ok := v.(api.ATMSupport); ok {
			if v, err := as.RetrieveTool(owner, kit); err != nil {
				continue
			} else {
				content = []byte(v.Content)
				break
			}
		} else if as, ok := v.(api.AssetFS); ok {
			if v, err := as.ReadFile(path.Join("tools", kit+".yaml")); err != nil {
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

	tc, err := LoadToolData([][]byte{content})
	if err != nil {
		return nil, err
	}
	if tc == nil || (len(tc.Tools) == 0) {
		return nil, fmt.Errorf("invalid config. no tools defined: %s", kit)
	}

	//
	tc.Kit = kit
	tc.RawContent = content

	return tc, nil
}

func (r *assetManager) ListModels(owner string) (map[string]*api.AppConfig, error) {
	var models = make(map[string]*api.AppConfig)
	for _, v := range r.assets {
		if as, ok := v.(api.ATMSupport); ok {
			if err := listModelsATM(owner, as, models); err != nil {
				return nil, err
			}
		} else if as, ok := v.(api.AssetFS); ok {
			if err := listModelsAsset(as, "models", models); err != nil {
				return nil, err
			}
		}
	}

	if len(models) == 0 {
		return nil, fmt.Errorf("no model configurations found")
	}

	return models, nil
}

func (r *assetManager) FindModels(owner string, set string) (*api.AppConfig, error) {
	var content []byte
	for _, v := range r.assets {
		if as, ok := v.(api.ATMSupport); ok {
			if v, err := as.RetrieveModel(owner, set); err != nil {
				// return nil, err
				continue
			} else {
				content = []byte(v.Content)
				break
			}
		} else if as, ok := v.(api.AssetFS); ok {
			if v, err := as.ReadFile(path.Join("models", set+".yaml")); err != nil {
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
		return nil, fmt.Errorf("failed to load models: %s. %v", set, err)
	}
	if mc == nil || len(mc.Models) == 0 {
		return nil, fmt.Errorf("invalid models config: %s", set)
	}

	//
	mc.Set = set
	mc.RawContent = content

	return mc, nil
}

// func ListAgents(assets api.AssetManager, user string) (string, int, error) {
// 	agents, err := assets.ListAgent(user)
// 	if err != nil {
// 		return "", 0, err
// 	}

// 	dict := make(map[string]*api.AgentConfig)
// 	for _, v := range agents {
// 		for _, sub := range v.Agents {
// 			dict[sub.Name] = sub
// 		}
// 	}

// 	keys := make([]string, 0)
// 	for k := range dict {
// 		keys = append(keys, k)
// 	}
// 	sort.Strings(keys)

// 	var buf strings.Builder
// 	for _, k := range keys {
// 		buf.WriteString(fmt.Sprintf("%s:\n    %s\n\n", k, dict[k].Description))
// 	}
// 	return buf.String(), len(keys), nil
// }

// func ListTools(assets api.AssetManager, user string) (string, int, error) {
// 	tools, err := assets.ListToolkit(user)
// 	if err != nil {
// 		return "", 0, err
// 	}

// 	list := []string{}
// 	for kit, tc := range tools {
// 		for _, v := range tc.Tools {
// 			// NOTE: Type in the output seems to confuse LLM (openai)
// 			list = append(list, fmt.Sprintf("%s:%s: %s\n", kit, v.Name, v.Description))
// 		}
// 	}

// 	sort.Strings(list)
// 	return strings.Join(list, "\n"), len(list), nil
// }

// func ListModels(assets api.AssetManager, user string) (string, int, error) {
// 	models, _ := assets.ListModels(user)

// 	list := []string{}
// 	for set, tc := range models {
// 		var keys []string
// 		for k := range tc.Models {
// 			keys = append(keys, k)
// 		}
// 		sort.Strings(keys)
// 		for _, level := range keys {
// 			v := tc.Models[level]
// 			list = append(list, fmt.Sprintf("%s/%s:\n    %s\n    %s\n    %s\n    %s\n", set, level, v.Provider, v.Model, v.BaseUrl, v.ApiKey))
// 		}
// 	}

// 	sort.Strings(list)
// 	return strings.Join(list, "\n"), len(list), nil
// }
