package hub

import (
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"

	"github.com/qiangli/ai/internal/log"
	"github.com/qiangli/ai/swarm/api"
)

type FileStore struct {
	Base string
}

func (r *FileStore) ReadDir(name string) ([]api.DirEntry, error) {
	return os.ReadDir(filepath.Join(r.Base, name))
}

func (r *FileStore) ReadFile(name string) ([]byte, error) {
	return os.ReadFile(filepath.Join(r.Base, name))
}

func (r *FileStore) read(name string) (string, []byte, error) {
	p := filepath.Join(r.Base, name)
	s, err := os.Stat(p)
	if err != nil {
		return "", nil, err
	}
	if s.IsDir() {
		entries, err := r.ReadDir(name)
		var dirs []*api.DirEntryInfo
		for _, v := range entries {
			if err != nil {
				return "", nil, err
			}
			dirs = append(dirs, api.FsDirEntryInfo(v))
		}
		data, err := json.Marshal(dirs)
		if err != nil {
			return "", nil, err
		}
		return "application/json", data, err
	}

	mimeTypes := map[string]string{
		"yaml": "text/yaml",
		"yml":  "text/yaml",
		"md":   "text/markdown",
	}
	ext := filepath.Ext(name)
	var contentType = "text/plain"
	if v, ok := mimeTypes[ext]; ok {
		contentType = v
	}
	data, err := r.ReadFile(name)
	return contentType, data, err
}

func createWebStoreHandler(base string) http.Handler {
	fs := FileStore{
		Base: base,
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log.Infof("web resource path: %v\n", r.URL.Path)

		if r.Method != http.MethodGet {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		contentType, data, err := fs.read(r.URL.Path)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte(err.Error()))
			return
		}
		w.Header().Set("Content-Type", contentType)
		w.WriteHeader(http.StatusOK)
		w.Write(data)
	})
}
