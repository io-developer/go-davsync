package model

import "io"

type Client interface {
	ReadTree() (paths []string, nodes map[string]Node, err error)

	ReadFile(path string) (io.Reader, error)

	AddDir(path string, recursive bool) error
	AddFile(path string, content io.Reader) error
}
