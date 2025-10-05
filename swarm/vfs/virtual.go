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
	ListDirectory(string) ([]string, error)
	CreateDirectory(string) error
	RenameFile(string, string) error
	GetFileInfo(string) (*FileInfo, error)
	ReadFile(string) ([]byte, error)
	WriteFile(string, []byte) error
	// EditFile
	SearchFiles(pattern string, path string, options *SearchOptions) (string, error)
}

type SearchOptions struct {
	// Parse PATTERN as a regular expression
	// Accepted syntax is the same
	// as https://github.com/google/re2/wiki/Syntax
	Regexp bool
	// Match case insensitively
	IgnoreCase bool
	// Only match whole words
	WordRegexp bool
	// Ignore files/directories matching pattern
	Exclude []string
	// Limit search to filenames matching PATTERN
	FileSearchRegexp string
	// Search up to 'Depth' directories deep (default: 25)
	Depth int
	// Follow symlinks
	Follow bool
	// Search hidden files and directories
	Hidden bool
}

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

type LocalFS struct {
	base string
}

func NewLocalFS(base string) FileSystem {
	return &LocalFS{
		base: base,
	}
}

func (s *LocalFS) ListDirectory(path string) ([]string, error) {
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

func (s *LocalFS) CreateDirectory(path string) error {
	validPath, err := s.validatePath(filepath.Join(s.base, path))
	if err != nil {
		return err
	}

	return os.MkdirAll(validPath, 0755)
}

func (s *LocalFS) RenameFile(source, destination string) error {
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

func (s *LocalFS) GetFileInfo(path string) (*FileInfo, error) {
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

func (s *LocalFS) ReadFile(path string) ([]byte, error) {
	validPath, err := s.validatePath(path)
	if err != nil {
		return nil, err
	}

	return os.ReadFile(validPath)
}

func (s *LocalFS) WriteFile(path string, content []byte) error {
	validPath, err := s.validatePath(path)
	if err != nil {
		return err
	}

	return os.WriteFile(validPath, content, 0644)
}

func (s *LocalFS) validatePath(path string) (string, error) {
	abs, err := filepath.Abs(path)
	if err != nil {
		return "", fmt.Errorf("invalid path %q: %w", path, err)
	}

	return abs, nil
}

func (s *LocalFS) getFileStats(path string) (*FileInfo, error) {
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

func (s *LocalFS) SearchFiles(pattern string, path string, options *SearchOptions) (string, error) {
	if options == nil {
		options = &SearchOptions{}
	}
	return Search(pattern, path, options)
}
