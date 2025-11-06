package conf

import (
	"path/filepath"

	"github.com/qiangli/ai/swarm"
	"github.com/qiangli/ai/swarm/api"
)

func NewBlobs(cfg *api.AppConfig, bucket string) (*swarm.BlobStorage, error) {
	res, err := api.LoadResourceConfig(filepath.Join(cfg.Base, "dhnt.json"))
	if err != nil {
		return nil, err
	}

	fs, err := swarm.NewCloudStorage(res)
	if err != nil {
		return nil, err
	}
	return swarm.NewBlobStorage(bucket, fs)
}
