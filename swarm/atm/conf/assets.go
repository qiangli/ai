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

func NewAssetManager(secrets api.SecretStore) api.AssetManager {
	return &assetManager{
		secrets: secrets,
		assets:  make([]api.AssetStore, 0),
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

// Return list of agent configs keyed by pack.
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

	if len(packs) == 0 {
		return nil, fmt.Errorf("no agent configurations found")
	}

	return packs, nil
	// var agents = make(map[string]*api.AppConfig)
	// // add sub agent to the map as well
	// for pack, v := range packs {
	// 	if len(v.Agents) == 0 {
	// 		continue
	// 	}
	// 	// Register the sub agent
	// 	for _, sub := range v.Agents {
	// 		key := pack + "/" + sub.Name
	// 		if _, exists := agents[key]; exists {
	// 			continue
	// 		}
	// 		agents[key] = v
	// 	}
	// }

	// if len(agents) == 0 {
	// 	return nil, fmt.Errorf("no agent configurations found")
	// }
	// return agents, nil
}

func (r *assetManager) FindAgent(owner string, pack string) (*api.AppConfig, error) {
	var content [][]byte
	var asset api.AssetStore
	var base string
	for _, v := range r.assets {
		if as, ok := v.(api.ATMSupport); ok {
			if v, err := as.RetrieveAgent(owner, pack); err != nil {
				continue
			} else {
				content = [][]byte{[]byte(v.Content)}
				asset = as
				base = ""
				break
			}
		} else if as, ok := v.(api.AssetFS); ok {
			// agents/<pack>.yaml
			if v, err := as.ReadFile(path.Join("agents", pack+".yaml")); err == nil {
				content = [][]byte{v}
				asset = as
				base = "agents"
				break
			}
			// agents/<pack>/*.yaml
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
						base = path.Join("agents", pack)
					}
				}
			}
			// found
			break
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

	//
	ac.Pack = pack

	if len(content) == 1 {
		ac.RawContent = content[0]
	} else {
		var buffer bytes.Buffer
		for _, part := range content {
			buffer.Write(part)
		}
		ac.RawContent = buffer.Bytes()
	}

	ac.Store = asset
	ac.BaseDir = base
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
	var content [][]byte
	var asset api.AssetStore
	var base string
	for _, v := range r.assets {
		if as, ok := v.(api.ATMSupport); ok {
			if v, err := as.RetrieveTool(owner, kit); err != nil {
				continue
			} else {
				content = [][]byte{[]byte(v.Content)}
				asset = as
				base = ""
				break
			}
		} else if as, ok := v.(api.AssetFS); ok {
			// tools/<kit>.yaml
			if v, err := as.ReadFile(path.Join("tools", kit+".yaml")); err == nil {
				content = [][]byte{v}
				asset = as
				base = "tools"
				break
			}
			// tools/<kit>/*.yaml
			dirs, err := as.ReadDir(path.Join("tools", kit))
			if err != nil {
				continue
			}
			for _, file := range dirs {
				if file.IsDir() {
					continue
				}
				name := file.Name()
				if strings.HasSuffix(name, ".yaml") || strings.HasSuffix(name, ".yml") {
					if v, err := as.ReadFile(path.Join("tools", kit, name)); err != nil {
						continue
					} else {
						content = append(content, v)
						asset = as
						base = path.Join("tools", kit)
					}
				}
			}
			// found
			break
		}
	}

	if len(content) == 0 {
		return nil, nil
	}

	tc, err := LoadToolData(content)
	if err != nil {
		return nil, err
	}
	if tc == nil || (len(tc.Tools) == 0) {
		return nil, fmt.Errorf("invalid config. no tools defined: %s", kit)
	}

	//
	tc.Kit = kit

	if len(content) == 1 {
		tc.RawContent = content[0]
	} else {
		var buffer bytes.Buffer
		for _, part := range content {
			buffer.Write(part)
		}
		tc.RawContent = buffer.Bytes()
	}

	tc.Store = asset
	tc.BaseDir = base
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
