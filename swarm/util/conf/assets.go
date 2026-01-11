package conf

import (
	"fmt"

	"github.com/qiangli/ai/swarm/api"
	"github.com/qiangli/ai/swarm/atm/conf"
	"github.com/qiangli/ai/swarm/resource"
)

// default assets with resource.json and the core
func Assets(cfg *api.DHNTConfig) (api.AssetManager, error) {
	var assets = conf.NewAssetManager()

	// cfg, err := api.LoadDHNTConfig(filepath.Join(base, "dhnt.json"))
	// if err != nil {
	// 	return nil, err
	// }

	for _, res := range cfg.Assets {
		switch res.Type {
		case "web":
			assets.AddStore(&resource.WebStore{
				Base:  fmt.Sprintf("%s/resource", res.Base),
				Token: res.Token,
			})
		case "file":
			assets.AddStore(&resource.FileStore{
				Base: res.Base,
			})
		default:
			return nil, fmt.Errorf("unsupported resource type: %s", res.Type)
		}
	}

	// defautl core
	assets.AddStore(resource.NewCoreStore())
	return assets, nil
}
