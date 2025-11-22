package atm

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/qiangli/ai/swarm/api"
	"github.com/qiangli/shell/tool/sh/vfs"
)

func (r *SystemKit) ListDirectory(ctx context.Context, vars *api.Vars, name string, args map[string]any) (string, error) {
	path, err := api.GetStrProp("path", args)
	if err != nil {
		return "", err
	}
	list, err := r.fs.ListDirectory(path)
	if err != nil {
		return "", err
	}
	return strings.Join(list, "\n"), nil
}

func (r *SystemKit) CreateDirectory(ctx context.Context, vars *api.Vars, name string, args map[string]any) (string, error) {
	path, err := api.GetStrProp("path", args)
	if err != nil {
		return "", err
	}
	return "", r.fs.CreateDirectory(path)
}

func (r *SystemKit) RenameFile(ctx context.Context, vars *api.Vars, name string, args map[string]any) (string, error) {
	source, err := api.GetStrProp("source", args)
	if err != nil {
		return "", err
	}
	dest, err := api.GetStrProp("destination", args)
	if err != nil {
		return "", err
	}
	if err := r.fs.RenameFile(source, dest); err != nil {
		return "", err
	}
	return "File renamed successfully", nil
}

func (r *SystemKit) GetFileInfo(ctx context.Context, vars *api.Vars, name string, args map[string]any) (string, error) {
	path, err := api.GetStrProp("path", args)
	if err != nil {
		return "", err
	}
	info, err := r.fs.FileInfo(path)
	if err != nil {
		return "", err
	}
	return info.String(), nil
}

// https://mimesniff.spec.whatwg.org/
// ReadFile returns mime type and the raw file content
func (r *SystemKit) ReadFile(ctx context.Context, vars *api.Vars, name string, args map[string]any) (any, error) {
	path, err := api.GetStrProp("path", args)
	if err != nil {
		return nil, err
	}
	raw, err := r.fs.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var c api.Blob
	c.Content = raw
	c.MimeType = http.DetectContentType(raw)

	return &c, nil
}

func (r *SystemKit) WriteFile(ctx context.Context, vars *api.Vars, name string, args map[string]any) (string, error) {
	path, err := api.GetStrProp("path", args)
	if err != nil {
		return "", err
	}
	content, err := api.GetStrProp("content", args)
	if err != nil {
		return "", err
	}
	if err := r.fs.WriteFile(path, []byte(content)); err != nil {
		return "", err
	}
	return "File written successfully", nil
}

func (r *SystemKit) EditFile(ctx context.Context, vars *api.Vars, name string, args map[string]any) (string, error) {
	path, err := api.GetStrProp("path", args)
	if err != nil {
		return "", err
	}

	// options
	find, err := api.GetStrProp("find", args)
	if err != nil {
		return "", err
	}
	replace, err := api.GetStrProp("replace", args)
	if err != nil {
		return "", err
	}
	all, err := api.GetBoolProp("all_occurrences", args)
	if err != nil {
		all = true
	}
	regex, err := api.GetBoolProp("regex", args)
	if err != nil {
		all = true
	}

	options := &vfs.EditOptions{
		Find:           find,
		Replace:        replace,
		AllOccurrences: all,
		UseRegex:       regex,
	}
	replacementCount, err := r.fs.EditFile(path, options)
	return fmt.Sprintf("File modified successfully. Made %d replacement(s).", replacementCount), nil
}

func (r *SystemKit) SearchFiles(ctx context.Context, vars *api.Vars, name string, args map[string]any) (string, error) {
	path, err := api.GetStrProp("path", args)
	if err != nil {
		return "", err
	}

	// options
	pattern, err := api.GetStrProp("pattern", args)
	if err != nil {
		return "", err
	}
	depth, err := api.GetIntProp("depth", args)
	if err != nil {
		depth = 5
	}
	exclude, _ := api.GetArrayProp("exclude", args)

	options := &vfs.SearchOptions{
		Pattern:    pattern,
		Regexp:     true,
		IgnoreCase: true,
		WordRegexp: false,
		Exclude:    exclude,
		Depth:      depth,
		Follow:     false,
		Hidden:     true,
	}
	return r.fs.SearchFiles(path, options)
}
