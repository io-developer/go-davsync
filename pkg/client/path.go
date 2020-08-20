package client

import (
	"path/filepath"
	"strings"
)

func NormalizePath(path string, isDir bool) string {
	norm := filepath.Join("/", strings.Trim(path, "/"))
	if isDir {
		norm += "/"
	}
	return norm
}
