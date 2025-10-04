package swarm

import (
	"fmt"

	"github.com/google/uuid"

	"github.com/qiangli/ai/swarm/api"
)

// a simple in memory object store
type BlobStorage struct {
	blobs map[string]*api.Blob
}

func (r *BlobStorage) Put(ID string, blob *api.Blob) error {
	r.blobs[ID] = blob
	return nil
}

func (r *BlobStorage) Get(ID string) (*api.Blob, error) {
	blob, exists := r.blobs[ID]
	if !exists {
		return nil, fmt.Errorf("blob with ID %s not found", ID)
	}
	return blob, nil
}

func (r *BlobStorage) List() ([]*api.Blob, error) {
	blobs := make([]*api.Blob, 0, len(r.blobs))
	for _, blob := range r.blobs {
		blobs = append(blobs, blob)
	}
	return blobs, nil
}

func NewBlobID() string {
	return uuid.NewString()
}

func NewBlobStorage() *BlobStorage {
	return &BlobStorage{
		blobs: make(map[string]*api.Blob),
	}
}
