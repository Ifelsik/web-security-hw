package fileutil

import (
	"fmt"
	"os"
)

// List files returns slice of all files in given dir.
func ListFiles(dir string) ([]string, error) {
	dirList, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("files list in dir: %w", err)
	}

	files := make([]string, 0, len(dirList))
	for _, item := range dirList {
		if item.IsDir() {
			continue
		}
		files = append(files, item.Name())
	}
	return files, nil
}
