package model

import (
	"os"
)

type Node struct {
	Name     string
	Path     string
	AbsPath  string
	IsDir    bool
	Size     int64
	FileInfo *os.FileInfo
	UserData interface{}
}

func (n Node) IsLocal() bool {
	return n.FileInfo != nil
}

func NodeComparePaths(from, to map[string]Node) (both, add, del []string) {
	both = []string{}
	add = []string{}
	del = []string{}
	for path := range from {
		if _, exists := to[path]; exists {
			both = append(both, path)
		} else {
			add = append(add, path)
		}
	}
	for path := range to {
		if _, exists := from[path]; !exists {
			del = append(del, path)
		}
	}
	return
}
