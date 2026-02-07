package gitkit

import "encoding/json"

func jsonMarshalImport(v any) ([]byte, error) {
	return json.Marshal(v)
}
