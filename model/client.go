package model

type Client interface {
	ReadTree() (paths []string, nodes map[string]Node, err error)

	Mkdir(path string, recursive bool) error
}
