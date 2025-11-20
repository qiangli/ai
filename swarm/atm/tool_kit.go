package atm

import (
	"fmt"
	"reflect"
	// "strconv"
	"strings"
)

// Default returns the given value if it's non-nil and non-zero value;
// otherwise, it returns the default value provided.
func Default(def, value any) any {
	v := reflect.ValueOf(value)
	if !v.IsValid() || reflect.DeepEqual(v.Interface(), reflect.Zero(v.Type()).Interface()) {
		return def
	}
	return value
}

// Spread concatenates the elements to create a single string.
func Spread(val any) string {
	if val == nil {
		return ""
	}
	var result = ""
	var items []string
	items, ok := val.([]string)
	if !ok {
		ar, ok := val.([]any)
		if ok {
			for _, v := range ar {
				if s, ok := v.(string); ok {
					items = append(items, s)
				} else {
					return fmt.Sprintf("%v", v)
				}
			}
		} else {
			return fmt.Sprintf("%v", val)
		}
	}

	for _, v := range items {
		if result != "" {
			result += " "
		}
		item := fmt.Sprintf("%v", v)
		// Escape double quotes and quote item if it contains spaces
		if strings.Contains(item, " ") {
			item = "\"" + strings.ReplaceAll(item, "\"", "\\\"") + "\""
		} else {
			item = strings.ReplaceAll(item, "\"", "\\\"")
		}

		result += item
	}
	return result
}

func CallKit(tool any, kit string, method string, args ...any) (any, error) {
	instance := reflect.ValueOf(tool)
	name := toPascalCase(method)
	m := instance.MethodByName(name)
	if !m.IsValid() {
		return nil, fmt.Errorf("method %s not found on %s", method, kit)
	}

	if m.Type().NumIn() != len(args) {
		return nil, fmt.Errorf("wrong number of arguments for %s.%s", kit, method)
	}

	in := make([]reflect.Value, len(args))
	for i, arg := range args {
		in[i] = reflect.ValueOf(arg)
	}
	results := m.Call(in)

	if len(results) < 2 {
		return nil, fmt.Errorf("unexpected number of return values for %s.%s", kit, method)
	}

	v := results[0].Interface()
	var err error
	if !results[1].IsNil() {
		err = results[1].Interface().(error)
	}

	return v, err
}

// // if required properties is not missing and is an array of strings
// // check if the required properties are present
// func IsRequired(key string, props map[string]any) bool {
// 	val, ok := props["required"]
// 	if !ok {
// 		return false
// 	}
// 	items, ok := val.([]string)
// 	if !ok {
// 		return false
// 	}
// 	for _, v := range items {
// 		if v == key {
// 			return true
// 		}
// 	}
// 	return false
// }

// func GetStrProp(key string, props map[string]any) (string, error) {
// 	val, ok := props[key]
// 	if !ok {
// 		if IsRequired(key, props) {
// 			return "", fmt.Errorf("missing property: %s", key)
// 		}
// 		return "", nil
// 	}
// 	str, ok := val.(string)
// 	if !ok {
// 		return "", fmt.Errorf("property '%s' must be a string", key)
// 	}
// 	return str, nil
// }

// func GetIntProp(key string, props map[string]any) (int, error) {
// 	val, ok := props[key]
// 	if !ok {
// 		if IsRequired(key, props) {
// 			return 0, fmt.Errorf("missing property: %s", key)
// 		}
// 		return 0, nil
// 	}
// 	switch v := val.(type) {
// 	case int:
// 		return v, nil
// 	case int32:
// 		return int(v), nil
// 	case int64:
// 		return int(v), nil
// 	case float32:
// 		return int(v), nil
// 	case float64:
// 		return int(v), nil
// 	default:
// 		s := fmt.Sprintf("%v", val)
// 		return strconv.Atoi(s)
// 	}
// }

// func GetArrayProp(key string, props map[string]any) ([]string, error) {
// 	val, ok := props[key]
// 	if !ok {
// 		if IsRequired(key, props) {
// 			return nil, fmt.Errorf("missing property: %s", key)
// 		}
// 		return []string{}, nil
// 	}
// 	items, ok := val.([]any)
// 	if ok {
// 		strs := make([]string, len(items))
// 		for i, v := range items {
// 			str, ok := v.(string)
// 			if !ok {
// 				return nil, fmt.Errorf("%s must be an array of strings", key)
// 			}
// 			strs[i] = str
// 		}
// 		return strs, nil
// 	}

// 	strs, ok := val.([]string)
// 	if !ok {
// 		if IsRequired(key, props) {
// 			return nil, fmt.Errorf("%s must be an array of strings", key)
// 		}
// 		return []string{}, nil
// 	}
// 	return strs, nil
// }

// func GetBoolProp(key string, props map[string]any) (bool, error) {
// 	val, ok := props[key]
// 	if !ok {
// 		if IsRequired(key, props) {
// 			return false, fmt.Errorf("missing property: %s", key)
// 		}
// 		return false, nil
// 	}
// 	if v, ok := val.(bool); ok {
// 		return v, nil
// 	}

// 	str, ok := val.(string)
// 	if !ok {
// 		return false, fmt.Errorf("property '%s' must be a boolean or a string representing a boolean: true or false", key)
// 	}
// 	switch str {
// 	case "true":
// 		return true, nil
// 	case "false":
// 		return false, nil
// 	default:
// 		return false, fmt.Errorf("property '%s' is a string but not a valid boolean value", key)
// 	}
// }
