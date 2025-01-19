package llm

import (
	"fmt"

	"github.com/openai/openai-go"
)

var allTools = append(append(aiHelpTools, systemTools...), dbTools...)

// if required properties is not missing and is an array of strings
// check if the required properties are present
func isRequired(key string, props map[string]interface{}) bool {
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

func getStrProp(key string, props map[string]interface{}) (string, error) {
	val, ok := props[key]
	if !ok {
		if isRequired(key, props) {
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

func getArrayProp(key string, props map[string]interface{}) ([]string, error) {
	val, ok := props[key]
	if !ok {
		if isRequired(key, props) {
			return nil, fmt.Errorf("missing property: %s", key)
		}
		return []string{}, nil
	}
	items, ok := val.([]interface{})
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
		if isRequired(key, props) {
			return nil, fmt.Errorf("%s must be an array of strings", key)
		}
		return []string{}, nil
	}
	return strs, nil
}

func filteredTools(names []string) []openai.ChatCompletionToolParam {
	var tools []openai.ChatCompletionToolParam
	for _, tool := range allTools {
		for _, name := range names {
			if tool.Function.Value.Name.Value == name {
				tools = append(tools, tool)
			}
		}
	}
	return tools
}
