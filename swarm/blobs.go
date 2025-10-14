package swarm

import (
	"encoding/json"
	"fmt"
	"path"

	"github.com/google/uuid"

	"github.com/qiangli/ai/swarm/api"
	"github.com/qiangli/ai/swarm/vfs"
)

type BlobStorage struct {
	bucket string
	fs     vfs.FileStore
}

func (r *BlobStorage) Put(ID string, blob *api.Blob) error {
	if err := r.fs.WriteFile(path.Join(r.bucket, ID), blob.Content); err != nil {
		return err
	}
	meta, err := json.Marshal(api.Blob{
		ID:       blob.ID,
		MimeType: blob.MimeType,
		Meta:     blob.Meta,
	})
	if err != nil {
		return nil
	}
	if err := r.fs.WriteFile(path.Join(r.bucket, ID+".json"), meta); err != nil {
		return err
	}
	return nil
}

func (r *BlobStorage) Get(ID string) (*api.Blob, error) {
	content, err := r.fs.ReadFile(path.Join(r.bucket, ID))
	if err != nil {
		return nil, fmt.Errorf("error reading blob content: %w", err)
	}

	metaData, err := r.fs.ReadFile(path.Join(r.bucket, ID+".json"))
	if err != nil {
		return nil, fmt.Errorf("error reading blob metadata: %w", err)
	}

	var meta api.Blob
	if err := json.Unmarshal(metaData, &meta); err != nil {
		return nil, fmt.Errorf("error unmarshaling blob metadata: %w", err)
	}

	return &api.Blob{
		ID:       meta.ID,
		MimeType: meta.MimeType,
		Meta:     meta.Meta,
		Content:  content,
	}, nil
}

func NewBlobID() string {
	return uuid.NewString()
}

func NewBlobStorage(bucket string, fs vfs.FileStore) *BlobStorage {
	return &BlobStorage{
		bucket: bucket,
		fs:     fs,
	}
}
