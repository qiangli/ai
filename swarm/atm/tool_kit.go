package atm

import (
	"fmt"
	"reflect"
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
