package explore

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

func extension(path string) string {
	return strings.TrimLeft(strings.ToLower(filepath.Ext(path)), ".")
}

func fileInfo(path string) (os.FileInfo, error) {
	fi, err := os.Stat(path)
	if err != nil {
		return nil, err
	}
	return fi, nil
}

func lookup(names []string, val string) string {
	for _, name := range names {
		val, ok := os.LookupEnv(name)
		if ok && val != "" {
			return val
		}
	}
	return val
}

func remove(path string) {
	fmt.Printf("this feature is disabled")
	//TODO
	// go func() {
	// 	cmd, ok := os.LookupEnv("WALK_REMOVE_CMD")
	// 	if !ok {
	// 		_ = os.RemoveAll(path)
	// 	} else {
	// 		_ = exec.Command(cmd, path).Run()
	// 	}
	// }()
}

func leaveOnlyAscii(content []byte) string {
	var result []byte

	for _, b := range content {
		if b == '\t' {
			result = append(result, ' ', ' ', ' ', ' ')
		} else if b == '\r' {
			continue
		} else if (b >= 32 && b <= 127) || b == '\n' { // '\n' is kept if newline needs to be retained
			result = append(result, b)
		}
	}

	return string(result)
}

func permBit(bit fs.FileMode, c byte) byte {
	if bit != 0 {
		return c
	}
	return '-'
}

func sortByModTime(a, b fs.DirEntry) bool {
	infoA, errA := a.Info()
	infoB, errB := b.Info()
	if errA != nil || errB != nil {
		// fallback to considering them equal if their info cannot be retrieved
		return false
	}

	if infoA.IsDir() && infoB.IsDir() {
		return infoA.ModTime().After(infoB.ModTime())
	}
	if infoA.IsDir() && !infoB.IsDir() {
		return true
	}
	if !infoA.IsDir() && infoB.IsDir() {
		return false
	}
	return infoA.ModTime().After(infoB.ModTime())
}

func sortByFilename(a, b fs.DirEntry) bool {
	if a.IsDir() && b.IsDir() {
		return a.Name() < b.Name()
	}
	if a.IsDir() && !b.IsDir() {
		return true
	}
	if !a.IsDir() && b.IsDir() {
		return false
	}
	return a.Name() < b.Name()
}

func sortByFileSize(a, b fs.DirEntry) bool {
	aIsDir := a.IsDir()
	bIsDir := b.IsDir()
	if aIsDir && bIsDir {
		return a.Name() < b.Name()
	}
	if aIsDir && !bIsDir {
		return true
	}
	if !aIsDir && bIsDir {
		return false
	}
	aInfo, _ := a.Info()
	bInfo, _ := b.Info()
	return aInfo.Size() < bInfo.Size()
}
