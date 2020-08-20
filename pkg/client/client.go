package client

import "io"

type Client interface {
	ReadTree() (paths []string, nodes map[string]Resource, err error)

	MakeDir(path string, recursive bool) error
	MakeDirFor(filePath string) error

	ReadFile(path string) (reader io.ReadCloser, err error)
	WriteFile(path string, content io.ReadCloser) error
	MoveFile(srcPath, dstPath string) error
}