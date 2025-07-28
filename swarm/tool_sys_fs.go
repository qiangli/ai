package swarm

import (
	"context"
	"net/http"
	"strings"

	"github.com/qiangli/ai/swarm/api"
	"github.com/qiangli/ai/swarm/vfs"
)

func (r *SystemKit) ListDirectory(ctx context.Context, vars *api.Vars, name string, args map[string]any) (string, error) {
	path, err := r.getStr("path", args)
	if err != nil {
		return "", err
	}
	list, err := _fs.ListDirectory(path)
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
	return "", _fs.CreateDirectory(path)
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
	if err := _fs.RenameFile(source, dest); err != nil {
		return "", err
	}
	return "File renamed successfully", nil
}

func (r *SystemKit) GetFileInfo(ctx context.Context, vars *api.Vars, name string, args map[string]any) (string, error) {
	path, err := r.getStr("path", args)
	if err != nil {
		return "", err
	}
	info, err := _fs.GetFileInfo(path)
	if err != nil {
		return "", err
	}
	return info.String(), nil
}

// https://mimesniff.spec.whatwg.org/
type FileContent struct {
	MimeType string
	Content  []byte

	// Optional message to LLM
	Message string
}

// ReadFile returns mime type and the raw file content
func (r *SystemKit) ReadFile(ctx context.Context, vars *api.Vars, name string, args map[string]any) (*FileContent, error) {
	path, err := r.getStr("path", args)
	if err != nil {
		return nil, err
	}
	raw, err := _fs.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var c FileContent
	c.Content = raw
	c.MimeType = http.DetectContentType(raw)
	c.Message = "File read successfully."

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
	if err := _fs.WriteFile(path, []byte(content)); err != nil {
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
	return _fs.SearchFiles(pattern, path, options)
}
