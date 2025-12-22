package api

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/url"
	"path/filepath"
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
		if v == nil {
			return nil
		}
		if len(v.Content) == 0 {
			return v
		}
		return &Result{
			MimeType: v.MimeType,
			Value:    MimeToString(v.MimeType, v.Content),
		}
	}
	if v, ok := data.(*Blob); ok {
		if v == nil {
			return nil
		}
		return &Result{
			MimeType: v.MimeType,
			Value:    MimeToString(v.MimeType, v.Content),
		}
	}
	if v, err := json.Marshal(data); err == nil {
		return &Result{
			Value: string(v),
		}
	}
	return &Result{
		Value: fmt.Sprintf("%v", data),
	}
}

func ToError(v any) error {
	if err, ok := v.(error); ok {
		return err
	}
	return fmt.Errorf("Error: %v", v)
}

// https://developer.mozilla.org/en-US/docs/Web/URI/Reference/Schemes/data
// data:[<media-type>][;base64],<data>
func DataURL(mime string, raw []byte) string {
	if mime == "" {
		// assume text
		return fmt.Sprintf("data:,%s", string(raw))
	}
	encoded := base64.StdEncoding.EncodeToString(raw)
	d := fmt.Sprintf("data:%s;base64,%s", mime, encoded)
	return d
}

// DecodeDataURL decodes a data URL and extracts the data as a string.
// It supports optional media types and base64 encoding, ensuring the "data:" prefix is present.
func DecodeDataURL(dataURL string) (string, error) {
	if !strings.HasPrefix(dataURL, "data:") {
		return dataURL, nil
	}

	dataURL = dataURL[5:]

	// data:,<content>
	if dataURL[0] == ',' {
		return dataURL[1:], nil
	}

	commaIndex := strings.Index(dataURL, ",")
	if commaIndex == -1 {
		return dataURL, nil
	}

	metadata := dataURL[:commaIndex]
	content := dataURL[commaIndex+1:]

	isBase64 := strings.HasSuffix(metadata, ";base64")

	// If the data is base64-encoded, decode it
	if isBase64 {
		decoded, err := base64.StdEncoding.DecodeString(content)
		if err != nil {
			return "", fmt.Errorf("failed to decode base64 data: %v", err)
		}
		return string(decoded), nil
	}
	return content, nil
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
		return MimeToString(v.MimeType, v.Content)
	}
	if v, ok := data.(*Blob); ok {
		return MimeToString(v.MimeType, v.Content)
	}
	if v, err := json.Marshal(data); err == nil {
		return string(v)
	}
	return fmt.Sprintf("%+v", data)
}

func MimeToString(mime string, content []byte) string {
	if mime == ContentTypeImageB64 {
		return string(content)
	}
	if strings.HasPrefix(mime, "text/") {
		return string(content)
	}
	return DataURL(mime, content)
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

func resolvePaths(dirs []string) ([]string, error) {
	uniquePaths := make(map[string]struct{})

	for _, dir := range dirs {
		realPath, err := filepath.EvalSymlinks(dir)
		if err != nil {
			// Handle error, for example by logging
			// log.Printf("Failed to evaluate symlink for %s: %v\n", dir, err)
			// continue
			return nil, err
		}
		uniquePaths[dir] = struct{}{}
		uniquePaths[realPath] = struct{}{}
	}

	var result []string
	for path := range uniquePaths {
		result = append(result, path)
	}

	return result, nil
}

func ToMessages(data any) []*Message {
	if data == nil || data == "" {
		return nil
	}
	if v, ok := data.([]*Message); ok {
		return v
	}
	if v, ok := data.(*Message); ok {
		return []*Message{v}
	}
	if v, ok := data.(string); ok {
		var ms []*Message
		if err := json.Unmarshal([]byte(v), &ms); err == nil {
			return ms
		}
		return []*Message{
			{
				Content: v,
			},
		}
	}
	return []*Message{
		{
			Content: fmt.Sprintf("%+v", data),
		},
	}
}

// Load data from uri.
// Support file:// and data: protocols
// TODO enforce protocol requiremnt?
// If no protocol is specified, assumse local file path.
func LoadURIContent(ws Workspace, uri string) (string, error) {
	if strings.HasPrefix(uri, "data:") {
		return DecodeDataURL(uri)
	} else {
		var f = uri
		if strings.HasPrefix(f, "file:") {
			v, err := url.Parse(f)
			if err != nil {
				return "", err
			}
			f = v.Path
		}
		data, err := ws.ReadFile(f, nil)
		if err != nil {
			return "", err
		}
		return string(data), nil
	}
}

func Cat(a, b, sep string) string {
	if a != "" && b == "" {
		return a
	} else if a == "" && b != "" {
		return b
	} else if a != "" && b != "" {
		return a + sep + b
	}
	return ""
}

// Check if strings starts with '#!' or length <= 120 and contains '{{'
// #! for multi-line large block of text
// {{ for oneliner
func IsTemplate(s string) bool {
	if strings.HasPrefix(s, "#!") {
		return true
	}
	if len(s) > 120 {
		return false
	}
	return strings.Contains(s, "{{")
}

// Check if ':' exists within the first 8 characters
// e.g:
// data:,
// file:///
// https://
func IsURI(s string) bool {
	endIndex := min(len(s), 8)
	return strings.Contains(s[:endIndex], ":")
}
