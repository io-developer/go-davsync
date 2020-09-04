package client

import "io"

type Client interface {
	ToAbsPath(relPath string) string
	ToRelativePath(absPath string) string

	ReadTree() (parents map[string]Resource, children map[string]Resource, err error)
	ReadResource(path string) (res Resource, exists bool, err error)

	MakeDir(path string) error
	MakeDirAbs(path string) error

	ReadFile(path string) (reader io.ReadCloser, err error)
	WriteFile(path string, content io.ReadCloser, size int64) error
	MoveFile(srcPath, dstPath string) error
	DeleteFile(path string) error
}
