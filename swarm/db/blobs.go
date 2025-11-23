package db

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"path"

	"github.com/google/uuid"

	"github.com/qiangli/ai/swarm/api"
	"github.com/qiangli/shell/tool/sh/vfs"
)

type BlobStorage struct {
	bucket string
	fs     vfs.FileStore
}

func (r *BlobStorage) Presign(ID string) (string, error) {
	// TODO the locator is presigned, should be done here?
	return r.fs.Locator(ID)
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
	content, err := r.fs.ReadFile(path.Join(r.bucket, ID), nil)
	if err != nil {
		return nil, fmt.Errorf("error reading blob content: %w", err)
	}

	metaData, err := r.fs.ReadFile(path.Join(r.bucket, ID+".json"), nil)
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

func NewBlobStorage(bucket string, fs vfs.FileStore) (*BlobStorage, error) {
	return &BlobStorage{
		bucket: bucket,
		fs:     fs,
	}, nil
}

type CloudStorage struct {
	Base  string
	Token string
}

// generate presigned url given the key
func (r CloudStorage) Locator(key string) (string, error) {
	endpoint := fmt.Sprintf("%s/blobs/presign?key=%s", r.Base, key)
	req, err := http.NewRequest(http.MethodGet, endpoint, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", "Bearer "+r.Token)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("failed to read blob %s: %s", key, resp.Status)
	}

	b, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	var v = struct {
		Token string
	}{}
	if err := json.Unmarshal(b, &v); err != nil {
		return "", err
	}
	locator := fmt.Sprintf("%s/cloud/locator/%s", r.Base, v.Token)
	return locator, nil
}

func (r CloudStorage) ReadFile(key string, o *vfs.ReadOptions) ([]byte, error) {
	endpoint := fmt.Sprintf("%s/blobs/file?key=%s", r.Base, key)
	req, err := http.NewRequest(http.MethodGet, endpoint, nil)
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
		return nil, fmt.Errorf("failed to read blob %s: %s", key, resp.Status)
	}

	if o == nil {
		return io.ReadAll(resp.Body)
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	//
	var content []string
	scanner := bufio.NewScanner(bytes.NewReader(data))
	for scanner.Scan() {
		content = append(content, scanner.Text())
	}

	if len(content) == 0 {
		return nil, fmt.Errorf("empty content")
	}

	startIdx := o.Offset
	endIdx := startIdx + o.Limit
	if startIdx >= len(content) {
		return nil, fmt.Errorf("error: line offset %d exceeds file length (%d lines)", o.Offset, len(content))
	}

	if endIdx > len(content) {
		endIdx = len(content)
	}

	selectedLines := content[startIdx:endIdx]
	lines := vfs.FormatLinesWithLineNumbers(selectedLines, startIdx+1)

	return []byte(lines), nil
}

func (r CloudStorage) WriteFile(key string, data []byte) error {
	endpoint := fmt.Sprintf("%s/blobs/file?key=%s", r.Base, key)
	req, err := http.NewRequest(http.MethodPut, endpoint, bytes.NewBuffer(data))
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
		return fmt.Errorf("failed to write blob %s: %s", key, resp.Status)
	}

	return nil
}

func NewCloudStorage(cfg *api.ResourceConfig) (vfs.FileStore, error) {
	return &CloudStorage{
		Base:  cfg.Base,
		Token: cfg.Token,
	}, nil
}
