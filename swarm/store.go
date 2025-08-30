package swarm

import (
	"embed"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"

	"github.com/qiangli/ai/swarm/api"
)

type ResourceStore struct {
	Base string
	FS   embed.FS
}

func (rs *ResourceStore) ReadDir(name string) ([]api.DirEntry, error) {
	p := fmt.Sprintf("%s/%s", rs.Base, name)
	return rs.FS.ReadDir(p)
}

func (rs *ResourceStore) ReadFile(name string) ([]byte, error) {
	p := fmt.Sprintf("%s/%s", rs.Base, name)
	return rs.FS.ReadFile(p)
}

func (rs *ResourceStore) Resolve(dir, name string) string {
	return fmt.Sprintf("%s/%s", dir, name)
}

type FileStore struct {
	Base string
}

func (fs *FileStore) ReadDir(name string) ([]api.DirEntry, error) {
	p := filepath.Join(fs.Base, name)
	return os.ReadDir(p)
}

func (fs *FileStore) ReadFile(name string) ([]byte, error) {
	p := filepath.Join(fs.Base, name)
	return os.ReadFile(p)
}

func (fs *FileStore) Resolve(dir, name string) string {
	return filepath.Join(dir, name)
}

type WebStore struct {
	Base  string
	Token string
}

func (ws *WebStore) ReadDir(name string) ([]api.DirEntry, error) {
	// Construct the URL
	url := fmt.Sprintf("%s/%s", ws.Base, name)

	// Make the HTTP GET request
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	// Add Authorization header
	req.Header.Add("Authorization", "Bearer "+ws.Token)
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode == 404 {
		return nil, nil
	}

	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("%s", resp.Status)
	}
	defer resp.Body.Close()

	// Decode the JSON response
	var data []*api.DirEntryInfo
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return nil, err
	}
	// required array conversion
	var entries []api.DirEntry
	for _, v := range data {
		entries = append(entries, v)
	}

	return entries, nil
}

func (ws *WebStore) ReadFile(name string) ([]byte, error) {
	// Construct the URL
	url := fmt.Sprintf("%s/%s", ws.Base, name)

	// Make the HTTP GET request
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	// Add Authorization header
	req.Header.Add("Authorization", "Bearer "+ws.Token)
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}

	// not found
	if resp.StatusCode == 404 {
		return nil, nil
	}

	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("%s", resp.Status)
	}
	defer resp.Body.Close()

	// Read the response body
	content, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	return content, nil
}

func (ws *WebStore) Resolve(base, name string) string {
	return fmt.Sprintf("%s/%s", base, name)
}
