package client

import "strings"

func NormalizePath(path string, isDir bool) string {
	norm := "/" + strings.Trim(path, "/")
	if isDir {
		norm += "/"
	}
	return norm
}
