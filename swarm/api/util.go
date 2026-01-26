package api

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/url"
	"path/filepath"
	"reflect"
	"slices"
	"strconv"
	"strings"

	"github.com/u-root/u-root/pkg/shlex"
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

	return ToStringArray(val), nil
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

func GetMapProp(key string, props map[string]any) (map[string]any, error) {
	val, ok := props[key]
	if !ok {
		if IsRequired(key, props) {
			return nil, fmt.Errorf("missing property: %s", key)
		}
		return map[string]any{}, nil
	}
	if v, ok := val.(map[string]any); ok {
		return v, nil
	}
	if v, ok := val.(map[string]string); ok {
		m := make(map[string]any)
		for key, value := range v {
			m[key] = value
		}
		return m, nil
	}
	if v, ok := val.(string); ok {
		m := make(map[string]any)
		if err := json.Unmarshal([]byte(v), &m); err != nil {
			return nil, err
		}
		return m, nil
	}
	return nil, fmt.Errorf("%q must be a JSON object map of environment variables, represented as key-value pairs", key)
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
	if len(dataURL) == 0 {
		return "", nil
	}

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
	// primitive
	switch v := data.(type) {
	case bool:
		return strconv.FormatBool(v)
	case int, int8, int16, int32, int64:
		return strconv.FormatInt(reflect.ValueOf(v).Int(), 10)
	case float32, float64:
		return strconv.FormatFloat(reflect.ValueOf(v).Float(), 'f', -1, 64)
	case string:
		return v
	}
	//
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

var TemplateMimeTypes = []string{"text/x-go-template", "x-go-template", "template", "tpl"}

// check
// #! or // magic for large block of text
// {{ contained within for oneliner
func IsTemplate(v any) bool {
	s, ok := v.(string)
	if !ok {
		return false
	}
	if strings.HasPrefix(s, "#!") || strings.HasPrefix(s, "//") {
		_, mime := ParseMimeType(s)
		return slices.Contains(TemplateMimeTypes, mime)
	}
	return strings.Contains(s, "{{")
}

// Check for mime-type specification.
// Returns content and mime type. remove first line for multiline text
func ParseMimeType(s string) (string, string) {
	var line string
	var data string
	parts := strings.SplitN(s, "\n", 2)
	// remove first line for multiline text
	if len(parts) == 2 {
		line = parts[0]
		data = parts[1]
	} else {
		line = parts[0]
		if len(line) > 256 {
			line = line[:256]
		}
		data = parts[0]
	}
	// shlex returns nil if not trimmed
	line = strings.TrimPrefix(line, "//")
	line = strings.TrimSpace(line)
	line = strings.TrimPrefix(line, "#!")
	opts := []string{"--mime-type", "--mime_type", "mime-type", "mime_type"}
	args := shlex.Argv(line)
	for i, v := range args {
		if slices.Contains(opts, v) {
			if len(args) > i+1 {
				return data, args[i+1]
			}
		}
		sa := strings.SplitN(v, "=", 2)
		if len(sa) == 1 {
			continue
		}
		if slices.Contains(opts, sa[0]) {
			return data, sa[1]
		}
	}
	return s, ""
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

// return an array of strings.
// try unmarshal obj if it is encoded in json format.
// return an empty array if obj is neither an array nor json array.
func ToStringArray(obj any) []string {
	if v, ok := obj.([]string); ok {
		return v
	}
	if v, ok := obj.([]any); ok {
		var sa []string
		for _, s := range v {
			sa = append(sa, ToString(s))
		}
		return sa
	}
	// json array
	if v, ok := obj.(string); ok {
		var sa []string
		if err := json.Unmarshal([]byte(v), &sa); err == nil {
			return sa
		}
		sa = ParseStringArray(v)
		if len(sa) > 0 {
			return sa
		}
		return []string{v}
	}
	return []string{}
}

// try parse string into an array of string in the following formats:
// a,b,c...
// [a,b,c,]
// intended for parsing commandline actions and model list
func ParseStringArray(s string) []string {
	unquote := func(x string) string {
		if x[0] == '\'' {
			return strings.Trim(x, "'")
		}
		if x[0] == '"' {
			return strings.Trim(x, "\"")
		}
		return strings.TrimSpace(x)
	}
	s = strings.TrimSpace(s)
	s = strings.TrimLeft(s, "[")
	s = strings.TrimRight(s, "]")
	pa := strings.Split(s, ",")
	var sa []string
	for _, v := range pa {
		v = strings.TrimSpace(v)
		if len(v) > 0 {
			sa = append(sa, unquote(v))
		}
	}
	return sa
}

// Abbreviate trims the string, keeping the beginning and end if exceeding maxLen.
// after replacing newlines with space
func Abbreviate(s string, maxLen int) string {
	if s == "" {
		return ""
	}
	// s = strings.ReplaceAll(s, "\n", "•")
	s = strings.ReplaceAll(s, "\n", " ")
	s = strings.Join(strings.Fields(s), " ")
	s = strings.TrimSpace(s)

	if len(s) > maxLen {
		// Calculate the length for each part
		keepLen := (maxLen - 3) / 2
		start := s[:keepLen]
		end := s[len(s)-keepLen:]
		return start + "…" + end
	}
	return s
}

func NilSafe[T any](ptr *T) T {
	var zeroValue T
	if ptr != nil {
		return *ptr
	}
	return zeroValue
}

func ToMap(obj any) (map[string]any, error) {
	if obj == nil {
		return nil, nil
	}
	if v, ok := obj.(map[string]any); ok {
		return v, nil
	}

	props := make(map[string]any)
	data, err := json.Marshal(obj)
	if err != nil {
		return nil, err
	}
	if err := json.Unmarshal(data, &props); err != nil {
		return nil, err
	}
	return props, nil
}
