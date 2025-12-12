package api

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"slices"
	"strconv"
	"strings"
)

// if required properties is not missing and is an array of strings
// check if the required properties are present
func IsRequired(key string, props map[string]any) bool {
	val, ok := props["required"]
	if !ok {
		return false
	}
	items, ok := val.([]string)
	if !ok {
		return false
	}
	return slices.Contains(items, key)
}

func GetStrProp(key string, props map[string]any) (string, error) {
	val, ok := props[key]
	if !ok {
		if IsRequired(key, props) {
			return "", fmt.Errorf("missing property: %s", key)
		}
		return "", nil
	}
	str, ok := val.(string)
	if !ok {
		return "", fmt.Errorf("property '%s' must be a string", key)
	}
	return str, nil
}

func GetIntProp(key string, props map[string]any) (int, error) {
	val, ok := props[key]
	if !ok {
		if IsRequired(key, props) {
			return 0, fmt.Errorf("missing property: %s", key)
		}
		return 0, nil
	}
	switch v := val.(type) {
	case int:
		return v, nil
	case int32:
		return int(v), nil
	case int64:
		return int(v), nil
	case float32:
		return int(v), nil
	case float64:
		return int(v), nil
	default:
		s := fmt.Sprintf("%v", val)
		return strconv.Atoi(s)
	}
}

func GetArrayProp(key string, props map[string]any) ([]string, error) {
	val, ok := props[key]
	if !ok {
		if IsRequired(key, props) {
			return nil, fmt.Errorf("missing property: %s", key)
		}
		return []string{}, nil
	}
	items, ok := val.([]any)
	if ok {
		strs := make([]string, len(items))
		for i, v := range items {
			str, ok := v.(string)
			if !ok {
				return nil, fmt.Errorf("%s must be an array of strings", key)
			}
			strs[i] = str
		}
		return strs, nil
	}

	strs, ok := val.([]string)
	if !ok {
		if IsRequired(key, props) {
			return nil, fmt.Errorf("%s must be an array of strings", key)
		}
		return []string{}, nil
	}
	return strs, nil
}

func GetBoolProp(key string, props map[string]any) (bool, error) {
	val, ok := props[key]
	if !ok {
		if IsRequired(key, props) {
			return false, fmt.Errorf("missing property: %s", key)
		}
		return false, nil
	}
	if v, ok := val.(bool); ok {
		return v, nil
	}

	str, ok := val.(string)
	if !ok {
		return false, fmt.Errorf("property '%s' must be a boolean or a string representing a boolean: true or false", key)
	}
	switch str {
	case "true":
		return true, nil
	case "false":
		return false, nil
	default:
		return false, fmt.Errorf("property '%s' is a string but not a valid boolean value", key)
	}
}

func ToResult(data any) *Result {
	if data == nil {
		return nil
	}
	if s, ok := data.(string); ok {
		return &Result{
			Value: s,
		}
	}
	if v, ok := data.(*Result); ok {
		if len(v.Content) == 0 {
			return v
		}
		return &Result{
			MimeType: v.MimeType,
			Value:    mimeToString(v.MimeType, v.Content),
		}
	}
	if v, ok := data.(*Blob); ok {
		return &Result{
			MimeType: v.MimeType,
			Value:    mimeToString(v.MimeType, v.Content),
		}
	}
	if v, err := json.Marshal(data); err == nil {
		return &Result{
			Value: string(v),
		}
	}
	return &Result{
		Value: fmt.Sprintf("%+v", data),
	}
}

// https://developer.mozilla.org/en-US/docs/Web/URI/Reference/Schemes/data
// data:[<media-type>][;base64],<data>
func dataURL(mime string, raw []byte) string {
	encoded := base64.StdEncoding.EncodeToString(raw)
	d := fmt.Sprintf("data:%s;base64,%s", mime, encoded)
	return d
}

func ToString(data any) string {
	if data == nil {
		return ""
	}
	if v, ok := data.(string); ok {
		return v
	}
	if v, ok := data.(*Result); ok {
		if len(v.Content) == 0 {
			return v.Value
		}
		return mimeToString(v.MimeType, v.Content)
	}
	if v, ok := data.(*Blob); ok {
		return mimeToString(v.MimeType, v.Content)
	}
	if v, err := json.Marshal(data); err == nil {
		return string(v)
	}
	return fmt.Sprintf("%+v", data)
}

func mimeToString(mime string, content []byte) string {
	if mime == ContentTypeImageB64 {
		return string(content)
	}
	if strings.HasPrefix(mime, "text/") {
		return string(content)
	}
	return dataURL(mime, content)
}

func ToInt(data any) int {
	if data == nil {
		return 0
	}
	if i, ok := data.(int); ok {
		return i
	}
	if i, ok := data.(int8); ok {
		return int(i)
	}
	if i, ok := data.(int16); ok {
		return int(i)
	}
	if i, ok := data.(int32); ok {
		return int(i)
	}
	if i, ok := data.(int64); ok {
		return int(i)
	}
	if s, ok := data.(string); ok {
		i, err := strconv.Atoi(s)
		if err == nil {
			return i
		}
	}
	return 0
}
