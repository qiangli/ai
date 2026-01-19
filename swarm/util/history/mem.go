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

func NewFileMemStore(workspace string) (api.MemStore, error) {
	base := filepath.Join(workspace, "history")
	if err := os.MkdirAll(base, 0755); err != nil {
		return nil, err
	}

	return &FileMemStore{
		base: base,
	}, nil
}

func (r *FileMemStore) Save(messages []*api.Message) error {
	return StoreHistory(r.base, messages)
}

func (r *FileMemStore) Load(opt *api.MemOption) ([]*api.Message, error) {
	return loadHistory(r.base, opt.MaxHistory, opt.MaxSpan, opt.Offset, opt.Roles)
}

func (r *FileMemStore) Get(id string) (*api.Message, error) {
	// TODO search
	max := 500
	span := 14400

	list, err := loadHistory(r.base, max, 0, span, nil)
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

func loadHistory(base string, maxHistory, maxSpan, offset int, roles []string) ([]*api.Message, error) {
	if maxHistory <= 0 || maxSpan <= 0 {
		return nil, nil
	}
	if offset < 0 {
		offset = 0
	}

	var history []*api.Message
	targetSize := offset + maxHistory
	messageCount := 0

	entries, err := os.ReadDir(base)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
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
			if err == nil && info.ModTime().After(old) {
				files = append(files, fileInfo{name: fullPath, mod: info.ModTime()})
			}
		}
	}

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

		// Collect messages from oldest to newest
		for i := len(msgs) - 1; i >= 0; i-- {
			msg := msgs[i]
			if msg.Context || (roles != nil && !slice.ContainsAny(roles, msg.Role)) {
				continue
			}
			if msg.ContentType != "" && !strings.HasPrefix(msg.ContentType, "text/") {
				continue
			}

			if messageCount < targetSize {
				history = append(history, msg)
				messageCount++
			}

			if messageCount >= targetSize {
				break
			}
		}

		if messageCount >= targetSize {
			break
		}
	}

	//
	if offset < len(history) {
		history = history[offset:]
	} else {
		history = []*api.Message{}
	}

	if len(history) > maxHistory {
		history = history[:maxHistory]
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
	now := time.Now()
	filename := fmt.Sprintf("%s-%d.json", now.Format("2006-01-02"), now.UnixNano())
	path := filepath.Join(base, filename)

	data, err := json.MarshalIndent(messages, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(path, data, 0644)
}
