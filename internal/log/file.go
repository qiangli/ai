package log

import (
	"os"
	"path/filepath"
)

type FileWriter struct {
	writer *os.File
}

func (cw *FileWriter) Write(p []byte) (n int, err error) {
	return cw.writer.Write(p)
}

func NewFileWriter(pathname string) (*FileWriter, error) {
	logDir := filepath.Dir(pathname)
	if logDir != "" {
		if err := os.MkdirAll(logDir, 0755); err != nil {
			return nil, err
		}
	}
	f, err := os.OpenFile(pathname, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		return nil, err
	}
	return &FileWriter{writer: f}, nil
}

func (cw *FileWriter) Close() error {
	if cw.writer == nil {
		return nil
	}
	return cw.writer.Close()
}
