package conf

import (
	"fmt"
	"path/filepath"

	"github.com/qiangli/ai/swarm/api"
	"github.com/qiangli/ai/swarm/atm/conf"
	"github.com/qiangli/ai/swarm/atm/resource"
)

// default assets with resource.json and standard
func Assets(cfg *api.AppConfig) (api.AssetManager, error) {
	var assets = conf.NewAssetManager()
	res, err := api.LoadResourceConfig(filepath.Join(cfg.Base, "dhnt.json"))
	if err != nil {
		return nil, err
	}
	assets.AddStore(&resource.WebStore{
		Base:  fmt.Sprintf("%s/resource", res.Base),
		Token: res.Token,
	})

	assets.AddStore(resource.NewStandardStore())
	return assets, nil
}
