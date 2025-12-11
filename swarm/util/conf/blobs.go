package conf

import (
	"path/filepath"

	"github.com/qiangli/ai/swarm/api"
	"github.com/qiangli/ai/swarm/db"
)

func NewBlobs(base string, bucket string) (*db.BlobStorage, error) {
	cfg, err := api.LoadDHNTConfig(filepath.Join(base, "dhnt.json"))
	if err != nil {
		return nil, err
	}

	fs, err := db.NewCloudStorage(cfg.Blob)
	if err != nil {
		return nil, err
	}
	return db.NewBlobStorage(bucket, fs)
}
