package conf

import (
	"encoding/json"

	"github.com/qiangli/ai/swarm/api"
)

func ListHistory(store api.MemStore, opt *api.MemOption) (string, int, error) {
	history, err := store.Load(opt)
	if err != nil {
		return "", 0, err
	}

	count := len(history)
	if count == 0 {
		return "", 0, api.NewNotFoundError("no messages")
	}

	b, err := json.MarshalIndent(history, "", "    ")
	if err != nil {
		return "", 0, err
	}
	return string(b), count, nil
}
