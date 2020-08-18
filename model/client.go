package model

import (
	"io"
)

type Client interface {
	ReadTree() (paths []string, nodes map[string]Node, err error)

	MakeDir(path string, recursive bool) error

	ReadFile(path string) (reader io.ReadCloser, err error)
	WriteFile(path string, content io.ReadCloser) error
}
