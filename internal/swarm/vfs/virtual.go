package vfs

import (
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// FileSystem is a virtual file system that provides a set of operations
// to interact with the file system in a controlled manner.
type FileSystem interface {
	// ListRoots() ([]string, error)
	ListDirectory(string) ([]string, error)
	CreateDirectory(string) error
	RenameFile(string, string) error
	GetFileInfo(string) (*FileInfo, error)
	ReadFile(string) ([]byte, error)
	WriteFile(string, []byte) error
}

// const (
// 	ListRootsToolName       = "list_roots"
// 	ListDirectoryToolName   = "list_directory"
// 	CreateDirectoryToolName = "create_directory"
// 	RenameFileToolName      = "rename_file"
// 	GetFileInfoToolName     = "get_file_info"
// 	ReadFileToolName        = "read_file"
// 	WriteFileToolName       = "write_file"
// 	// TempDirToolName         = "temp_dir"
// )

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

type VirtualFS struct {
	// roots []string
}

// func NewVFS() (*VirtualFS) {
// 	s := &VirtualFS{}
// 	return s
// }

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
		prefix := "File"
		if entry.IsDir() {
			prefix = "Direcotory"
		}
		result = append(result, fmt.Sprintf("%s: %s", prefix, entry.Name()))
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

// func (s *VirtualFS) isTemp(path string) bool {
// 	if strings.HasPrefix(path, s.TempDir()) {
// 		return true
// 	}
// 	if strings.HasPrefix(path, "/tmp") {
// 		return true
// 	}
// 	if strings.HasPrefix(path, os.Getenv("TMPDIR")) {
// 		return true
// 	}
// 	return false
// }

func (s *VirtualFS) validatePath(path string) (string, error) {
	// // always allow temp directories
	// if s.isTemp(path) {
	// 	return path, nil
	// }

	abs, err := filepath.Abs(path)
	if err != nil {
		return "", fmt.Errorf("invalid path %q: %w", path, err)
	}

	return abs, nil

	// for _, root := range s.roots {
	// 	if strings.HasPrefix(abs, root) {
	// 		return abs, nil
	// 	}
	// }
	// return "", fmt.Errorf("access denied - path outside allowed directories: %s", abs)
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
