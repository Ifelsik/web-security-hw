package fileutil

import (
	"path/filepath"
	"strings"
)

// Filename returns name of file without extension and dir path.
// For example, "/mnt/my_disk/amelia.jpg" will be converted to "amelia".
func Filename(path string) string {
	_, fileExt := filepath.Split(path)
	ext := filepath.Ext(fileExt)
	file := strings.Split(fileExt, ext)
	return file[0]
}
