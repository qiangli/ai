package hub

import (
	"encoding/json"
	"fmt"

	"github.com/qiangli/ai/internal/log"
)

type KVPayload struct {
	Bucket string `json:"bucket"`

	Key   string `json:"key"`
	Value string `json:"value"`

	// bucket: create/drop key-value: set/get/remove
	Action string `json:"action"`
}

func (h *Hub) doKV(data string) (ActionStatus, string) {
	log.Debugf("hub doKV: %v", data)

	var payload KVPayload
	if err := json.Unmarshal([]byte(data), &payload); err != nil {
		return StatusError, err.Error()
	}

	//
	var bucket = payload.Bucket
	var key = payload.Key
	var val = payload.Value

	if bucket == "" {
		return StatusError, "missing bucket name"
	}
	switch payload.Action {
	case "create":
		if err := h.kvStore.Create(bucket); err != nil {
			return StatusError, err.Error()
		}
		return StatusOK, fmt.Sprintf("bucket %q created", bucket)
	case "drop":
		if err := h.kvStore.Drop(bucket); err != nil {
			return StatusError, err.Error()
		}
		return StatusOK, fmt.Sprintf("bucket %q removed", bucket)
	}

	if key == "" {
		return StatusError, "missing key"
	}
	switch payload.Action {
	case "set":
		err := h.kvStore.Put(bucket, key, val)
		if err != nil {
			return StatusError, err.Error()
		}
		return StatusOK, "Done"
	case "get":
		val, err := h.kvStore.Get(bucket, key)
		if err != nil {
			return StatusError, err.Error()
		}
		if val == nil {
			return StatusNotFound, fmt.Sprintf("Not found: %s in %s", key, bucket)
		}
		return StatusOK, *val
	case "remove":
		err := h.kvStore.Delete(bucket, key)
		if err != nil {
			return StatusError, err.Error()
		}
		return StatusOK, "Done"
	}

	return StatusError, fmt.Sprintf("Action not supported: %s", payload.Action)
}
