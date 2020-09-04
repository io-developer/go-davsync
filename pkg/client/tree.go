package client

import (
	"sort"
	"strings"

	"github.com/io-developer/go-davsync/pkg/util"
)

type Tree interface {
	ReadTree() (parents map[string]Resource, children map[string]Resource, err error)
	GetResource(path string) (res Resource, exists bool, err error)
}

type TreeBuffer struct {
	client      Client
	isReaden    bool
	parents     map[string]Resource
	children    map[string]Resource
	createdDirs map[string]string
}

func NewTreeBuffer(client Client) *TreeBuffer {
	return &TreeBuffer{
		client:      client,
		createdDirs: make(map[string]string),
	}
}

func (t *TreeBuffer) ToAbsPath(relPath string) string {
	return t.client.ToAbsPath(relPath)
}

func (t *TreeBuffer) ToRelativePath(absPath string) string {
	return t.client.ToRelativePath(absPath)
}

func (t *TreeBuffer) Read() (err error) {
	t.parents, t.children, err = t.client.ReadTree()
	if err != nil {
		t.isReaden = true
		t.createdDirs = make(map[string]string)
	}
	return
}

func (t *TreeBuffer) readIfNeeded() error {
	if t.isReaden {
		return nil
	}
	return t.Read()
}

func (t *TreeBuffer) GetParents() map[string]Resource {
	return t.parents
}

func (t *TreeBuffer) GetParent(path string) (r Resource, exists bool) {
	r, exists = t.parents[path]
	return
}

func (t *TreeBuffer) GetParentPaths() []string {
	paths := make([]string, len(t.parents))
	for path := range t.parents {
		paths = append(paths, path)
	}
	sort.Slice(paths, func(i, j int) bool {
		return paths[i] < paths[j]
	})
	return paths
}

func (t *TreeBuffer) GetChildren() map[string]Resource {
	return t.children
}

func (t *TreeBuffer) GetChild(path string) (r Resource, exists bool) {
	r, exists = t.children[path]
	return
}

func (t *TreeBuffer) GetChildrenPaths() []string {
	paths := make([]string, len(t.children))
	for path := range t.children {
		paths = append(paths, path)
	}
	sort.Slice(paths, func(i, j int) bool {
		return paths[i] < paths[j]
	})
	return paths
}

func (t *TreeBuffer) MakeDir(path string, recursive bool) error {
	if recursive {
		return t.makeDirRecursive(t.ToAbsPath(path))
	}
	return t.makeDir(t.ToAbsPath(path))
}

func (t *TreeBuffer) MakeDirAbs(absPath string, recursive bool) error {
	if recursive {
		return t.makeDirRecursive(absPath)
	}
	return t.makeDir(absPath)
}

func (t *TreeBuffer) makeDirRecursive(absPath string) error {
	parts := strings.Split(strings.Trim(absPath, "/"), "/")
	total := len(parts)
	if total < 1 {
		return nil
	}
	subDir := "/"
	for _, part := range parts {
		if part != "" {
			subDir += part + "/"
			err := t.makeDir(subDir)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func (t *TreeBuffer) makeDir(absPath string) (err error) {
	err = t.readIfNeeded()
	if err != nil {
		return err
	}
	absPath = util.PathNormalize(absPath, true)
	if _, exists := t.parents[absPath]; exists {
		return nil
	}
	if _, exists := t.createdDirs[absPath]; exists {
		return nil
	}
	path := t.ToRelativePath(absPath)
	if item, exists := t.children[path]; exists && item.IsDir {
		return nil
	}
	err = t.client.MakeDirAbs(absPath)
	if err == nil {
		t.createdDirs[absPath] = absPath
	}
	return nil
}
