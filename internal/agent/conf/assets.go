package conf

import (
	"github.com/qiangli/ai/swarm/api"
	"github.com/qiangli/ai/swarm/atm/conf"
	"github.com/qiangli/ai/swarm/atm/resource"
)

func Assets(app *api.AppConfig) api.AssetManager {
	var assets = conf.NewAssetManager()
	if app.AgentResource != nil {
		for _, v := range app.AgentResource.Resources {
			assets.AddStore(&resource.WebStore{
				Base:  v.Base,
				Token: v.Token,
			})
		}
	}
	assets.AddStore(resource.NewStandardStore())
	return assets
}
