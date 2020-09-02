package util

import (
	"path/filepath"
	"regexp"
	"sort"
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

func PathSorted(paths []string) []string {
	sorted := []string{}
	for _, p := range paths {
		sorted = append(sorted, p)
	}
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i] < sorted[j]
	})
	return sorted
}

func PathSortedDirs(paths []string) []string {
	re := regexp.MustCompile("^.*/")
	dict := map[string]string{}
	for _, p := range paths {
		dir := re.FindString(p)
		if dir != "" {
			dict[dir] = dir
		}
	}
	sorted := []string{}
	for p := range dict {
		sorted = append(sorted, p)
	}
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i] < sorted[j]
	})
	return sorted
}
