package swarm

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
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

type CloudStorage struct {
	Token    string
	Endpoint string
}

func (r CloudStorage) ReadFile(key string) ([]byte, error) {
	url := fmt.Sprintf("%s?key=%s", r.Endpoint, key)
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+r.Token)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to read file: %s", resp.Status)
	}

	return io.ReadAll(resp.Body)
}

func (r CloudStorage) WriteFile(key string, data []byte) error {
	url := fmt.Sprintf("%s?key=%s", r.Endpoint, key)
	req, err := http.NewRequest(http.MethodPut, url, bytes.NewBuffer(data))
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+r.Token)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to write file: %s", resp.Status)
	}

	return nil
}

func NewCloudStorage(endpoint, token string) vfs.FileStore {
	return &CloudStorage{
		Endpoint: endpoint,
		Token:    token,
	}
}
