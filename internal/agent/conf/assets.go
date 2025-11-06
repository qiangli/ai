package conf

import (
	"fmt"
	// "os"
	"path/filepath"

	"github.com/qiangli/ai/swarm/api"
	"github.com/qiangli/ai/swarm/atm/conf"
	"github.com/qiangli/ai/swarm/atm/resource"
)

// default assets with resource.json and standard
func Assets(app *api.AppConfig) (api.AssetManager, error) {
	var assets = conf.NewAssetManager()

	// if ar, err := api.LoadAgentResource(filepath.Join(app.Base, "resource.json")); err == nil {
	// 	for _, v := range ar.Resources {
	// 		assets.AddStore(&resource.WebStore{
	// 			Base:  v.Base,
	// 			Token: v.Token,
	// 		})
	// 	}
	// }
	cfg, err := api.LoadResourceConfig(filepath.Join(app.Workspace, "dhnt.json"))
	if err != nil {
		return nil, err
	}
	assets.AddStore(&resource.WebStore{
		Base:  fmt.Sprintf("%s/resource", cfg.Base),
		Token: cfg.Token,
	})

	assets.AddStore(resource.NewStandardStore())
	return assets, nil
}
