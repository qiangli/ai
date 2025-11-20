package api

import (
	"fmt"
	"strconv"
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
	for _, v := range items {
		if v == key {
			return true
		}
	}
	return false
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
