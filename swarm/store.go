package swarm

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"

	"github.com/qiangli/ai/swarm/api"
)

type FileStore struct {
}

func (fs *FileStore) ReadDir(name string) ([]api.DirEntry, error) {
	return os.ReadDir(name)
}

func (fs *FileStore) ReadFile(name string) ([]byte, error) {
	return os.ReadFile(name)
}

type WebStore struct {
	Base string
}

func (r *WebStore) ReadDir(name string) ([]api.DirEntry, error) {
	// Construct the URL
	url := fmt.Sprintf("%s/%s", r.Base, name)

	// Make the HTTP GET request
	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	// Decode the JSON response
	var data []*api.DirEntryInfo
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return nil, err
	}
	var entries []api.DirEntry
	for _, v := range data {
		entries = append(entries, v)
	}

	return entries, nil
}

func (r *WebStore) ReadFile(name string) ([]byte, error) {
	// Construct the URL
	url := fmt.Sprintf("%s/%s", r.Base, name)

	// Make the HTTP GET request
	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	// Read the response body
	content, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	return content, nil
}
