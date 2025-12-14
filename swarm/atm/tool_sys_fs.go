package atm

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/qiangli/ai/swarm/api"
	"github.com/qiangli/shell/tool/sh/vfs"
)

func (r *SystemKit) ListRoots(ctx context.Context, vars *api.Vars, name string, args map[string]any) (string, error) {
	var result strings.Builder
	result.WriteString("Allowed Root Directories:\n\n")
	roots, err := vars.RTE.Roots.ResolveRoots()
	if err != nil {
		return "", fmt.Errorf("failed to resolve root directories: %v", roots)
	}
	if len(roots) == 0 {
		return "", fmt.Errorf("no accessible root directories")
	}
	v, err := PrettyJSON(roots)
	if err != nil {
		return "", err
	}
	result.WriteString(v)
	return result.String(), nil
}

func (r *SystemKit) ListDirectory(ctx context.Context, vars *api.Vars, name string, args map[string]any) (string, error) {
	path, err := api.GetStrProp("path", args)
	if err != nil {
		return "", err
	}
	list, err := vars.RTE.Workspace.ListDirectory(path)
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
	return "", vars.RTE.Workspace.CreateDirectory(path)
}

func (r *SystemKit) Tree(ctx context.Context, vars *api.Vars, name string, args map[string]any) (string, error) {
	path, err := api.GetStrProp("path", args)
	if err != nil {
		return "", err
	}

	depth := 3 // Default value
	if v, err := api.GetIntProp("depth", args); err == nil {
		depth = int(v)
	}

	// Extract follow_symlinks parameter (optional, default: false)
	followSymlinks := false // Default value
	if v, err := api.GetBoolProp("follow_symlinks", args); err == nil {
		followSymlinks = v
	}

	result, err := vars.RTE.Workspace.Tree(path, depth, followSymlinks)
	if err != nil {
		return "", err
	}
	return result, nil
}

func (r *SystemKit) GetFileInfo(ctx context.Context, vars *api.Vars, name string, args map[string]any) (string, error) {
	path, err := api.GetStrProp("path", args)
	if err != nil {
		return "", err
	}
	info, err := vars.RTE.Workspace.GetFileInfo(path)
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

	var opt *vfs.ReadOptions

	num, _ := api.GetBoolProp("number", args)
	offset, _ := api.GetIntProp("offset", args)
	limit, _ := api.GetIntProp("limit", args)
	if num || offset > 0 || limit > 0 {
		opt = &vfs.ReadOptions{
			Number: num,
			Offset: offset,
			Limit:  limit,
		}
	}

	raw, err := vars.RTE.Workspace.ReadFile(path, opt)
	if err != nil {
		return nil, err
	}

	var c api.Blob
	c.Content = raw
	c.MimeType = http.DetectContentType(raw)

	return &c, nil
}

func (r *SystemKit) ReadMultipleFiles(ctx context.Context, vars *api.Vars, name string, args map[string]any) (string, error) {
	paths, err := api.GetArrayProp("paths", args)
	if err != nil {
		return "", err
	}

	results, err := vars.RTE.Workspace.ReadMultipleFiles(paths)
	if err != nil {
		return "", err
	}
	return strings.Join(results, "\n"), nil
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
	if err := vars.RTE.Workspace.WriteFile(path, []byte(content)); err != nil {
		return "", err
	}
	return "File written successfully", nil
}

func (r *SystemKit) CopyFile(ctx context.Context, vars *api.Vars, name string, args map[string]any) (string, error) {
	source, err := api.GetStrProp("source", args)
	if err != nil {
		return "", err
	}
	dest, err := api.GetStrProp("destination", args)
	if err != nil {
		return "", err
	}
	if err := vars.RTE.Workspace.CopyFile(source, dest); err != nil {
		return "", err
	}
	return "File renamed successfully", nil
}

func (r *SystemKit) MoveFile(ctx context.Context, vars *api.Vars, name string, args map[string]any) (string, error) {
	source, err := api.GetStrProp("source", args)
	if err != nil {
		return "", err
	}
	dest, err := api.GetStrProp("destination", args)
	if err != nil {
		return "", err
	}
	if err := vars.RTE.Workspace.MoveFile(source, dest); err != nil {
		return "", err
	}
	return "File renamed successfully", nil
}

func (r *SystemKit) DeleteFile(ctx context.Context, vars *api.Vars, name string, args map[string]any) (string, error) {
	path, err := api.GetStrProp("path", args)
	if err != nil {
		return "", err
	}

	recursive, err := api.GetBoolProp("recursive", args)
	if err != nil {
		recursive = false
	}

	if err := vars.RTE.Workspace.DeleteFile(path, recursive); err != nil {
		return "", err
	}

	return fmt.Sprintf("Successfully deleted %q", path), nil
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
	replacementCount, err := vars.RTE.Workspace.EditFile(path, options)
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
	return vars.RTE.Workspace.SearchFiles(path, options)
}
