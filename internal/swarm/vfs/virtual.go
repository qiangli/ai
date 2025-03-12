package vfs

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// FileSystem is a virtual file system that provides a set of operations
// to interact with the file system in a controlled manner.
type FileSystem interface {
	ListRoots() ([]string, error)
	ListDirectory(string) ([]string, error)
	CreateDirectory(string) error
	RenameFile(string, string) error
	GetFileInfo(string) (*FileInfo, error)
	ReadFile(string) ([]byte, error)
	WriteFile(string, []byte) error
	// Describe(string) *Descriptor
	TempDir() string
}

const (
	ListRootsToolName       = "list_roots"
	ListDirectoryToolName   = "list_directory"
	CreateDirectoryToolName = "create_directory"
	RenameFileToolName      = "rename_file"
	GetFileInfoToolName     = "get_file_info"
	ReadFileToolName        = "read_file"
	WriteFileToolName       = "write_file"
	TempDirToolName         = "temp_dir"
)

type FileInfo struct {
	IsDirectory bool `json:"isDirectory"`
	IsFile      bool `json:"isFile"`

	Permissions string `json:"permissions"`

	Size     int64     `json:"size"`
	Created  time.Time `json:"created"`
	Modified time.Time `json:"modified"`
	Accessed time.Time `json:"accessed"`
}

func (f *FileInfo) String() string {
	return fmt.Sprintf(
		"IsDirectory: %t, IsFile: %t, Permissions: %s, Size: %d, Created: %s, Modified: %s, Accessed: %s",
		f.IsDirectory,
		f.IsFile,
		f.Permissions,
		f.Size,
		f.Created.Format(time.RFC3339),
		f.Modified.Format(time.RFC3339),
		f.Accessed.Format(time.RFC3339),
	)
}

// type Descriptor struct {
// 	Name        string
// 	Description string
// 	Parameters  map[string]any
// }

// var Descriptors = map[string]*Descriptor{
// 	ListRootsToolName: {
// 		Name:        ListRootsToolName,
// 		Description: "Returns the list of directories that this server is allowed to access.",
// 		Parameters: map[string]any{
// 			"type":       "object",
// 			"properties": map[string]any{},
// 		},
// 	},
// 	ListDirectoryToolName: {
// 		Name:        ListDirectoryToolName,
// 		Description: "Get a detailed listing of all files and directories in a specified path.",
// 		Parameters: map[string]any{
// 			"type": "object",
// 			"properties": map[string]any{
// 				"path": map[string]any{
// 					"type":        "string",
// 					"description": "Path of the directory to list",
// 				},
// 			},
// 			"required": []string{"path"},
// 		},
// 	},
// 	CreateDirectoryToolName: {
// 		Name:        CreateDirectoryToolName,
// 		Description: "Create a new directory or ensure a directory exists.",
// 		Parameters: map[string]any{
// 			"type": "object",
// 			"properties": map[string]any{
// 				"path": map[string]any{
// 					"type":        "string",
// 					"description": "Path of the directory to create",
// 				},
// 			},
// 			"required": []string{"path"},
// 		},
// 	},
// 	RenameFileToolName: {
// 		Name:        RenameFileToolName,
// 		Description: "Rename files and directories.",
// 		Parameters: map[string]any{
// 			"type": "object",
// 			"properties": map[string]any{
// 				"source": map[string]any{
// 					"type":        "string",
// 					"description": "Source path of the file or directory",
// 				},
// 				"destination": map[string]any{
// 					"type":        "string",
// 					"description": "Destination path",
// 				},
// 			},
// 			"required": []string{"source", "destination"},
// 		},
// 	},
// 	GetFileInfoToolName: {
// 		Name:        GetFileInfoToolName,
// 		Description: "Retrieve detailed metadata about a file or directory.",
// 		Parameters: map[string]any{
// 			"type": "object",
// 			"properties": map[string]any{
// 				"path": map[string]any{
// 					"type":        "string",
// 					"description": "Path to the file or directory",
// 				},
// 			},
// 			"required": []string{"path"},
// 		},
// 	},
// 	ReadFileToolName: {
// 		Name:        ReadFileToolName,
// 		Description: "Read the complete contents of a file from the file system.",
// 		Parameters: map[string]any{
// 			"type": "object",
// 			"properties": map[string]any{
// 				"path": map[string]any{
// 					"type":        "string",
// 					"description": "Path to the file to read",
// 				},
// 			},
// 			"required": []string{"path"},
// 		},
// 	},
// 	WriteFileToolName: {
// 		Name:        WriteFileToolName,
// 		Description: "Create a new file or overwrite an existing file with new content.",
// 		Parameters: map[string]any{
// 			"type": "object",
// 			"properties": map[string]any{
// 				"path": map[string]any{
// 					"type":        "string",
// 					"description": "Path where to write the file",
// 				},
// 				"content": map[string]any{
// 					"type":        "string",
// 					"description": "Content to write to the file",
// 				},
// 			},
// 			"required": []string{"path", "content"},
// 		},
// 	},
// 	TempDirToolName: {
// 		Name:        TempDirToolName,
// 		Description: "Return the default directory to use for temporary files",
// 		Parameters:  map[string]any{},
// 	},
// }

type VirtualFS struct {
	roots []string
}

func NewVFS(roots []string) (*VirtualFS, error) {
	normalized := make([]string, 0, len(roots))
	for _, dir := range roots {
		abs, err := filepath.Abs(dir)
		if err != nil {
			return nil, fmt.Errorf("failed to resolve path %s: %w", dir, err)
		}

		info, err := os.Stat(abs)
		if err != nil {
			return nil, fmt.Errorf(
				"failed to access directory %s: %w",
				abs,
				err,
			)
		}
		if !info.IsDir() {
			return nil, fmt.Errorf("path is not a directory: %s", abs)
		}

		normalized = append(normalized, filepath.Clean(abs))
	}

	s := &VirtualFS{
		roots: normalized,
	}

	return s, nil
}

// func (s *VirtualFS) Describe(name string) *Descriptor {
// 	if desc, ok := Descriptors[name]; ok {
// 		return desc
// 	}
// 	return nil
// }

func (s *VirtualFS) ListRoots() ([]string, error) {
	return s.roots, nil
}

func (s *VirtualFS) ListDirectory(path string) ([]string, error) {
	validPath, err := s.validatePath(path)
	if err != nil {
		return nil, err
	}

	entries, err := os.ReadDir(validPath)
	if err != nil {
		return nil, err
	}

	var result []string
	for _, entry := range entries {
		suffix := ""
		if entry.IsDir() {
			suffix = "/"
		}
		result = append(result, fmt.Sprintf("%s%s", entry.Name(), suffix))
	}

	return result, nil
}

func (s *VirtualFS) CreateDirectory(path string) error {
	validPath, err := s.validatePath(path)
	if err != nil {
		return err
	}

	return os.MkdirAll(validPath, 0755)
}

func (s *VirtualFS) RenameFile(source, destination string) error {
	validSource, err := s.validatePath(source)
	if err != nil {
		return err
	}
	validDest, err := s.validatePath(destination)
	if err != nil {
		return err
	}

	return os.Rename(validSource, validDest)
}

func (s *VirtualFS) GetFileInfo(path string) (*FileInfo, error) {
	validPath, err := s.validatePath(path)
	if err != nil {
		return nil, err
	}

	info, err := s.getFileStats(validPath)
	if err != nil {
		return nil, err
	}

	return info, nil
}

func (s *VirtualFS) ReadFile(path string) ([]byte, error) {
	validPath, err := s.validatePath(path)
	if err != nil {
		return nil, err
	}

	return os.ReadFile(validPath)
}

func (s *VirtualFS) WriteFile(path string, content []byte) error {
	validPath, err := s.validatePath(path)
	if err != nil {
		return err
	}

	return os.WriteFile(validPath, content, 0644)
}

func (s *VirtualFS) isTemp(path string) bool {
	if strings.HasPrefix(path, s.TempDir()) {
		return true
	}
	if strings.HasPrefix(path, "/tmp") {
		return true
	}
	if strings.HasPrefix(path, os.Getenv("TMPDIR")) {
		return true
	}
	return false
}

func (s *VirtualFS) validatePath(requestedPath string) (string, error) {
	// always allow temp directories
	if s.isTemp(requestedPath) {
		return requestedPath, nil
	}

	abs, err := filepath.Abs(requestedPath)
	if err != nil {
		return "", fmt.Errorf("invalid path: %w", err)
	}

	normalized := filepath.Clean(abs)

	allowed := false
	for _, dir := range s.roots {
		if strings.HasPrefix(normalized, dir) {
			allowed = true
			break
		}
	}
	if !allowed {
		return "", fmt.Errorf(
			"access denied - path outside allowed directories: %s",
			abs,
		)
	}

	realPath, err := filepath.EvalSymlinks(abs)
	if err != nil {
		if !os.IsNotExist(err) {
			return "", err
		}
		parent := filepath.Dir(abs)
		realParent, err := filepath.EvalSymlinks(parent)
		if err != nil {
			return "", fmt.Errorf("parent directory does not exist: %s", parent)
		}
		normalizedParent := filepath.Clean(realParent)
		for _, dir := range s.roots {
			if strings.HasPrefix(normalizedParent, dir) {
				return abs, nil
			}
		}
		return "", fmt.Errorf(
			"access denied - parent directory outside allowed directories",
		)
	}

	normalizedReal := filepath.Clean(realPath)
	for _, dir := range s.roots {
		if strings.HasPrefix(normalizedReal, dir) {
			return realPath, nil
		}
	}
	return "", fmt.Errorf(
		"access denied - symlink target outside allowed directories",
	)
}

func (s *VirtualFS) getFileStats(path string) (*FileInfo, error) {
	info, err := os.Stat(path)
	if err != nil {
		return &FileInfo{}, err
	}

	return &FileInfo{
		IsDirectory: info.IsDir(),
		IsFile:      !info.IsDir(),
		Permissions: fmt.Sprintf("%o", info.Mode().Perm()),
		Size:        info.Size(),
		Created:     info.ModTime(),
		Modified:    info.ModTime(),
		Accessed:    info.ModTime(),
	}, nil
}

func (s *VirtualFS) TempDir() string {
	return os.TempDir()
}
