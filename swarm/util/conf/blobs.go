package conf

import (
	"path/filepath"

	"github.com/qiangli/ai/swarm/api"
	"github.com/qiangli/ai/swarm/db"
)

func NewBlobs(cfg *api.AppConfig, bucket string) (*db.BlobStorage, error) {
	res, err := api.LoadResourceConfig(filepath.Join(cfg.Base, "dhnt.json"))
	if err != nil {
		return nil, err
	}

	fs, err := db.NewCloudStorage(res)
	if err != nil {
		return nil, err
	}
	return db.NewBlobStorage(bucket, fs)
}
