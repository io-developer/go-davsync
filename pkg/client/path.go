package client

import (
	"path/filepath"
	"strings"
)

func PathNormalize(path string, isDir bool) string {
	norm := filepath.Join("/", strings.Trim(path, "/"))
	if isDir && norm != "/" {
		norm += "/"
	}
	return norm
}

func PathNormalizeBaseDir(baseDir string) string {
	norm := filepath.Join("/", baseDir)
	return strings.TrimRight(norm, "/") + "/"
}

func PathRel(absPath, baseDir string) string {
	isDir := strings.HasSuffix(absPath, "/")
	rel := strings.TrimPrefix(
		PathNormalize(absPath, isDir),
		PathNormalizeBaseDir(baseDir),
	)
	return PathNormalize(rel, isDir)
}

func PathAbs(relPath, baseDir string) string {
	isDir := strings.HasSuffix(relPath, "/")
	path := filepath.Join(baseDir, relPath)
	return PathNormalize(path, isDir)
}

func PathParents(path string) []string {
	norm := filepath.Join("/", strings.Trim(path, "/"))
	parents := []string{}
	parent := ""
	parts := strings.Split(norm, "/")
	for i := 0; i < len(parts)-1; i++ {
		part := parts[i]
		parent += part + "/"
		parents = append(parents, parent)
	}
	return parents
}
