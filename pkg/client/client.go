package client

import "io"

type Client interface {
	Tree

	ToAbsPath(relPath string) string
	ToRelativePath(absPath string) string

	MakeDir(path string, recursive bool) error
	MakeDirFor(filePath string) error

	ReadFile(path string) (reader io.ReadCloser, err error)
	WriteFile(path string, content io.ReadCloser, size int64) error
	MoveFile(srcPath, dstPath string) error
	DeleteFile(path string) error
}
