package swarm

import (
	"bytes"
	"context"
	"encoding/base64"
	"fmt"
	"image"
	"net/http"
	"strings"

	"github.com/qiangli/ai/swarm/api"
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

type FileContent struct {
	MimeType string
	Content  []byte
}

func (r *SystemKit) ReadFile(ctx context.Context, vars *api.Vars, name string, args map[string]any) (*FileContent, error) {
	path, err := r.getStr("path", args)
	if err != nil {
		return nil, err
	}
	raw, err := _fs.ReadFile(path)
	if err != nil {
		return nil, err
	}

	isImage := func(data []byte) bool {
		reader := bytes.NewReader(data)
		img, _, err := image.DecodeConfig(reader)
		if err != nil {
			return false
		}
		return img.Width > 0 && img.Height > 0
	}

	var c FileContent
	c.Content = raw

	// TODO other types
	if isImage(raw) {
		c.MimeType = http.DetectContentType(raw)
	}

	return &c, nil
}

func (r *SystemKit) ReadEncodeFile(ctx context.Context, vars *api.Vars, name string, args map[string]any) (string, error) {
	path, err := r.getStr("path", args)
	if err != nil {
		return "", err
	}
	return readEncodeFile(path)
}

func readEncodeFile(p string) (string, error) {
	raw, err := _fs.ReadFile(p)
	if err != nil {
		return "", err
	}

	encoded := base64.StdEncoding.EncodeToString(raw)
	dataURL := fmt.Sprintf("data:image/png;base64,%s", encoded)

	return dataURL, nil
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

func (r *SystemKit) DecodeWriteFile(ctx context.Context, vars *api.Vars, name string, args map[string]any) (string, error) {
	path, err := r.getStr("path", args)
	if err != nil {
		return "", err
	}
	encoded, err := r.getStr("content", args)
	if err != nil {
		return "", err
	}
	if err := decodeWriteFile(path, encoded); err != nil {
		return "", err
	}
	return "File written successfully", nil
}

func decodeWriteFile(p, encoded string) error {
	content, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return err
	}
	if err := _fs.WriteFile(p, content); err != nil {
		return err
	}
	return nil
}
