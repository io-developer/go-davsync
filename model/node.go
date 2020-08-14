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
