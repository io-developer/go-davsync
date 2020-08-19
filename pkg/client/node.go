package client

import "os"

type Resource struct {
	Name     string
	Path     string
	AbsPath  string
	IsDir    bool
	Size     int64
	FileInfo *os.FileInfo
	UserData interface{}
}

func (n Resource) IsLocal() bool {
	return n.FileInfo != nil
}
