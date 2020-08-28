package client

type Tree interface {
	ReadParents() (absPaths []string, items map[string]Resource, err error)
	ReadTree() (paths []string, items map[string]Resource, err error)
	GetResource(path string) (res Resource, exists bool, err error)
}
