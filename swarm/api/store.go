package api

import (
	"encoding/json"
	"io/fs"
	"time"
)

// TODO
type MemOption struct {
	MaxHistory int
	MaxSpan    int
}

type MemStore interface {
	Save([]*Message) error
	Load(*MemOption) ([]*Message, error)
}

type DirEntry = fs.DirEntry

type DirEntryInfo struct {
	name  string
	isDir bool
	mode  uint32
	info  *FileInfo
}

func NewDirEntry(
	name string,
	isDir bool,
	mode uint32,
	info *FileInfo,
) DirEntry {
	return &DirEntryInfo{
		name:  name,
		isDir: isDir,
		mode:  mode,
		info:  info,
	}
}

func (r DirEntryInfo) MarshalJSON() ([]byte, error) {
	return json.Marshal(&struct {
		Name  string    `json:"name"`
		IsDir bool      `json:"isDir"`
		Mode  uint32    `json:"type"`
		Info  *FileInfo `json:"info"`
	}{
		Name:  r.name,
		IsDir: r.isDir,
		Mode:  r.mode,
		Info:  r.info,
	})
}

func (r *DirEntryInfo) UnmarshalJSON(data []byte) error {
	aux := &struct {
		Name  string    `json:"name"`
		IsDir bool      `json:"isDir"`
		Mode  uint32    `json:"type"`
		Info  *FileInfo `json:"info"`
	}{}

	if err := json.Unmarshal(data, aux); err != nil {
		return err
	}

	r.name = aux.Name
	r.isDir = aux.IsDir
	r.mode = aux.Mode
	r.info = aux.Info

	return nil
}

func FsDirEntryInfo(de fs.DirEntry) *DirEntryInfo {
	fi, _ := de.Info()
	return &DirEntryInfo{
		name:  de.Name(),
		isDir: de.IsDir(),
		mode:  uint32(de.Type()),
		info:  FsFileInfo(fi),
	}
}

func (r *DirEntryInfo) Name() string {
	return r.name
}

func (r *DirEntryInfo) IsDir() bool {
	return r.isDir
}

func (r *DirEntryInfo) Type() fs.FileMode {
	return fs.FileMode(r.mode)
}

func (r *DirEntryInfo) Info() (fs.FileInfo, error) {
	return r.info, nil
}

type FileInfo struct {
	name    string
	size    int64
	mode    uint32
	modTime time.Time
	isDir   bool
}

func NewFileInfo(
	name string,
	size int64,
	mode uint32,
	modTime time.Time,
	isDir bool,
) *FileInfo {
	return &FileInfo{
		name:    name,
		size:    size,
		mode:    mode,
		modTime: modTime,
		isDir:   isDir,
	}
}

func (fi FileInfo) MarshalJSON() ([]byte, error) {
	return json.Marshal(&struct {
		Name    string    `json:"name"`
		Size    int64     `json:"size"`
		Mode    uint32    `json:"mode"`
		ModTime time.Time `json:"modTime"`
		IsDir   bool      `json:"isDir"`
	}{
		Name:    fi.name,
		Size:    fi.size,
		Mode:    fi.mode,
		ModTime: fi.modTime,
		IsDir:   fi.isDir,
	})
}

func (fi *FileInfo) UnmarshalJSON(data []byte) error {
	var aux struct {
		Name    string    `json:"name"`
		Size    int64     `json:"size"`
		Mode    uint32    `json:"mode"`
		ModTime time.Time `json:"modTime"`
		IsDir   bool      `json:"isDir"`
	}

	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}

	fi.name = aux.Name
	fi.size = aux.Size
	fi.mode = aux.Mode
	fi.modTime = aux.ModTime
	fi.isDir = aux.IsDir
	return nil
}

func FsFileInfo(fi fs.FileInfo) *FileInfo {
	return &FileInfo{
		name:    fi.Name(),
		size:    fi.Size(),
		mode:    uint32(fi.Mode()),
		modTime: fi.ModTime(),
		isDir:   fi.IsDir(),
	}
}

func (r *FileInfo) Name() string {
	return r.name
}

func (r *FileInfo) Size() int64 {
	return r.size
}

func (r *FileInfo) Mode() fs.FileMode {
	return fs.FileMode(r.mode)
}

func (r *FileInfo) ModTime() time.Time {
	return r.modTime
}

func (r *FileInfo) IsDir() bool {
	return r.isDir
}

func (r *FileInfo) Sys() any {
	return nil
}

type AssetStore interface {
	ReadDir(name string) ([]DirEntry, error)
	ReadFile(name string) ([]byte, error)
	Resolve(parent string, name string) string
}
