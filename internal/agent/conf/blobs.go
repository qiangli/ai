package conf

import (
	"path/filepath"

	"github.com/qiangli/ai/swarm"
	"github.com/qiangli/ai/swarm/api"
)

func NewBlobs(app *api.AppConfig, bucket string) (*swarm.BlobStorage, error) {
	cfg, err := swarm.LoadCloudConfig(filepath.Join(app.Base, "dhnt.json"))
	if err != nil {
		return nil, err
	}

	fs, err := swarm.NewCloudStorage(cfg)
	if err != nil {
		return nil, err
	}
	return swarm.NewBlobStorage(bucket, fs)
}
