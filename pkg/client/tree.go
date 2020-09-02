package client

import (
	"regexp"
	"sort"
	"strings"

	"github.com/io-developer/go-davsync/pkg/util"
)

type Tree interface {
	ReadParents() (absPaths []string, items map[string]Resource, err error)
	ReadTree() (paths []string, items map[string]Resource, err error)
	GetResource(path string) (res Resource, exists bool, err error)
}

type TreeBuffer struct {
	client      Client
	parents     map[string]Resource
	items       map[string]Resource
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

func (t *TreeBuffer) GetParents() map[string]Resource {
	return t.parents
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

func (t *TreeBuffer) ReadParents() (absPaths []string, items map[string]Resource, err error) {
	if t.parents == nil {
		_, t.parents, err = t.client.ReadParents()
	}
	if err != nil {
		return
	}
	return t.GetParentPaths(), t.parents, err
}

func (t *TreeBuffer) GetTree() map[string]Resource {
	return t.items
}

func (t *TreeBuffer) GetTreePaths() []string {
	paths := make([]string, len(t.items))
	for path := range t.items {
		paths = append(paths, path)
	}
	sort.Slice(paths, func(i, j int) bool {
		return paths[i] < paths[j]
	})
	return paths
}

func (t *TreeBuffer) ReadTree() (paths []string, items map[string]Resource, err error) {
	if t.items == nil {
		_, t.items, err = t.client.ReadTree()
	}
	if err != nil {
		return
	}
	return t.GetTreePaths(), t.items, err
}

func (t *TreeBuffer) GetTreeResource(path string) (r Resource, exists bool) {
	r, exists = t.items[path]
	return
}

func (t *TreeBuffer) MakeDir(path string, recursive bool) error {
	if recursive {
		return t.makeDirRecursive(t.ToAbsPath(path))
	}
	return t.makeDir(t.ToAbsPath(path))
}

func (t *TreeBuffer) MakeDirFor(filePath string) error {
	re, err := regexp.Compile("(^|/+)[^/]+$")
	if err != nil {
		return err
	}
	dir := re.ReplaceAllString(filePath, "")
	return t.makeDirRecursive(t.ToAbsPath(dir))
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

func (t *TreeBuffer) makeDir(absPath string) error {
	absPath = util.PathNormalize(absPath, true)
	_, parents, err := t.ReadParents()
	if _, exists := parents[absPath]; exists {
		return nil
	}
	if _, exists := t.createdDirs[absPath]; exists {
		return nil
	}
	path := t.ToRelativePath(absPath)
	_, items, err := t.ReadTree()
	if err != nil {
		return err
	}
	if item, exists := items[path]; exists && item.IsDir {
		return nil
	}
	err = t.client.MakeDir(path, false)
	if err == nil {
		t.createdDirs[absPath] = absPath
	}
	return nil
}
