package atm

import (
	"context"
	"net/http"
	"strings"

	"github.com/qiangli/ai/swarm/api"
	// "github.com/qiangli/ai/swarm/vfs"
	"github.com/qiangli/shell/tool/sh/vfs"
)

func (r *SystemKit) ListDirectory(ctx context.Context, vars *api.Vars, name string, args map[string]any) (string, error) {
	path, err := r.getStr("path", args)
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
	path, err := r.getStr("path", args)
	if err != nil {
		return "", err
	}
	return "", r.fs.CreateDirectory(path)
}

func (r *SystemKit) RenameFile(ctx context.Context, vars *api.Vars, name string, args map[string]any) (string, error) {
	source, err := r.getStr("source", args)
	if err != nil {
		return "", err
	}
	dest, err := r.getStr("destination", args)
	if err != nil {
		return "", err
	}
	if err := r.fs.RenameFile(source, dest); err != nil {
		return "", err
	}
	return "File renamed successfully", nil
}

func (r *SystemKit) FileInfo(ctx context.Context, vars *api.Vars, name string, args map[string]any) (string, error) {
	path, err := r.getStr("path", args)
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
	path, err := r.getStr("path", args)
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
	path, err := r.getStr("path", args)
	if err != nil {
		return "", err
	}
	content, err := r.getStr("content", args)
	if err != nil {
		return "", err
	}
	if err := r.fs.WriteFile(path, []byte(content)); err != nil {
		return "", err
	}
	return "File written successfully", nil
}

func (r *SystemKit) SearchFiles(ctx context.Context, vars *api.Vars, name string, args map[string]any) (string, error) {
	pattern, err := r.getStr("pattern", args)
	if err != nil {
		return "", err
	}
	path, err := r.getStr("path", args)
	if err != nil {
		return "", err
	}
	// options
	exclude, err := r.getArray("exclude", args)
	if err != nil {
		return "", err
	}
	options := &vfs.SearchOptions{
		Regexp:     true,
		IgnoreCase: true,
		WordRegexp: false,
		Exclude:    exclude,
		Follow:     false,
		Hidden:     true,
	}
	return r.fs.SearchFiles(pattern, path, options)
}
