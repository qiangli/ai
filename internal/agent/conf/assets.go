package conf

import (
	"path/filepath"

	"github.com/qiangli/ai/swarm/api"
	"github.com/qiangli/ai/swarm/atm/conf"
	"github.com/qiangli/ai/swarm/atm/resource"
)

// default assets with resource.json and standard
func Assets(app *api.AppConfig) api.AssetManager {
	var assets = conf.NewAssetManager()

	if ar, err := api.LoadAgentResource(filepath.Join(app.Base, "resource.json")); err == nil {
		for _, v := range ar.Resources {
			assets.AddStore(&resource.WebStore{
				Base:  v.Base,
				Token: v.Token,
			})
		}
	}

	assets.AddStore(resource.NewStandardStore())
	return assets
}
