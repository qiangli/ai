package calllog

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/qiangli/ai/swarm/api"
)

type FileCallLog struct {
	base string
}

func (r *FileCallLog) Base() string {
	return r.base
}

func NewFileCallLog(root string, sessionID api.SessionID) (api.CallLogger, error) {
	sub := fmt.Sprintf("session_id__%s", sessionID)
	base := filepath.Join(root, "toolcall", sub)
	if err := os.MkdirAll(base, 0755); err != nil {
		return nil, err
	}

	return &FileCallLog{
		base: base,
	}, nil
}

func (r *FileCallLog) Save(entry *api.CallLogEntry) {
	write(r.base, entry)
}

func write(base string, entry *api.CallLogEntry) error {
	// filename
	now := time.Now()
	id := api.NewKitname(entry.Kit, entry.Name).ID()
	var filename string
	if entry.Error != nil {
		filename = fmt.Sprintf("%s-%s-%d-error.json", id, now.Format("2006-01-02"), now.UnixNano())
	} else {
		filename = fmt.Sprintf("%s-%s-%d.json", id, now.Format("2006-01-02"), now.UnixNano())
	}
	path := filepath.Join(base, filename)

	data, err := json.MarshalIndent(entry, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(path, data, 0644)
}
