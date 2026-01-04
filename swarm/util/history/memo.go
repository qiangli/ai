package history

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/charmbracelet/x/exp/slice"
	"github.com/qiangli/ai/swarm/api"
)

type FileMemStore struct {
	base string
}

func NewFileMemStore(workspace string) api.MemStore {
	return &FileMemStore{
		base: filepath.Join(workspace, "history"),
	}
}

func (r *FileMemStore) Save(messages []*api.Message) error {
	return StoreHistory(r.base, messages)
}

func (r *FileMemStore) Load(opt *api.MemOption) ([]*api.Message, error) {
	return LoadHistory(r.base, opt.MaxHistory, opt.MaxSpan, opt.Roles)
}

func (r *FileMemStore) Get(id string) (*api.Message, error) {
	// TODO search
	max := 100
	span := 14400

	list, err := LoadHistory(r.base, max, span, nil)
	if err != nil {
		return nil, err
	}
	for _, v := range list {
		if v.ID == id {
			return v, nil
		}
	}
	return nil, api.NewNotFoundError("message id: " + id)
}

func LoadHistory(base string, maxHistory, maxSpan int, roles []string) ([]*api.Message, error) {
	if maxHistory <= 0 || maxSpan <= 0 {
		return nil, nil
	}

	var history []*api.Message

	entries, err := os.ReadDir(base)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	// Collect .json files and their infos
	type fileInfo struct {
		name string
		mod  time.Time
	}
	var files []fileInfo

	old := time.Now().Add(-time.Duration(maxSpan) * time.Minute)

	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".json") {
			fullPath := filepath.Join(base, entry.Name())
			info, err := os.Stat(fullPath)
			if err == nil {
				if info.ModTime().Before(old) {
					continue
				}
				files = append(files, fileInfo{name: fullPath, mod: info.ModTime()})
			}
		}
	}

	// Sort by mod time DESC (most recent first)
	sort.Slice(files, func(i, j int) bool {
		return files[i].mod.After(files[j].mod)
	})

	for _, fi := range files {
		data, err := os.ReadFile(fi.name)
		if err != nil {
			continue
		}
		var msgs []*api.Message
		if err := json.Unmarshal(data, &msgs); err != nil {
			continue
		}
		for i := len(msgs) - 1; i >= 0; i-- {
			// only use text message for now
			for _, msg := range msgs {
				if roles != nil && !slice.ContainsAny(roles, msg.Role) {
					continue
				}
				if msg.ContentType == "" || strings.HasPrefix(msg.ContentType, "text/") {
					history = append(history, msg)
				}
			}

			if maxHistory > 0 && len(history) >= maxHistory {
				result := history[:maxHistory]
				reverseMessages(result)
				return result, nil
			}
		}
	}

	reverseMessages(history)
	return history, nil
}

func reverseMessages(msgs []*api.Message) {
	for left, right := 0, len(msgs)-1; left < right; left, right = left+1, right-1 {
		msgs[left], msgs[right] = msgs[right], msgs[left]
	}
}

func StoreHistory(base string, messages []*api.Message) error {
	if err := os.MkdirAll(base, 0755); err != nil {
		return err
	}

	// filename
	now := time.Now()
	filename := fmt.Sprintf("%s-%d.json", now.Format("2006-01-02"), now.UnixNano())
	path := filepath.Join(base, filename)

	data, err := json.MarshalIndent(messages, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(path, data, 0644)
}
