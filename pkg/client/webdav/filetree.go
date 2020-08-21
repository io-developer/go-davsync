package webdav

import (
	"strings"

	"github.com/io-developer/go-davsync/pkg/client"
)

type FileTree struct {
	opt       Options
	adapter   *Adapter
	parents   map[string]Propfind
	items     map[string]Propfind
	itemPaths []string
}

func NewFileTree(opt Options) *FileTree {
	return &FileTree{
		opt:     opt,
		adapter: NewAdapter(opt),
	}
}

func (c *FileTree) GetParents() (map[string]Propfind, error) {
	var err error
	if c.parents == nil {
		c.parents = make(map[string]Propfind)
		err = c.readParents()
	}
	return c.parents, err
}

func (c *FileTree) readParents() error {
	parts := strings.Split(strings.Trim(c.opt.BaseDir, "/"), "/")
	total := len(parts)
	if total < 1 {
		return nil
	}
	path := ""
	for _, part := range parts {
		path += "/" + part
		some, code, err := c.adapter.Propfind(path, "0")
		if code == 404 {
			return nil
		}
		if err != nil {
			return err
		}
		if len(some.Propfinds) < 1 {
			return err
		}
		normPath := client.PathNormalize(path, true)
		c.parents[normPath] = some.Propfinds[0]
	}
	return nil
}

func (c *FileTree) GetItems() (paths []string, items map[string]Propfind, err error) {
	if c.items == nil {
		c.itemPaths = []string{}
		c.items = map[string]Propfind{}
		err = c.readDir("/")
	}
	return c.itemPaths, c.items, err
}

func (c *FileTree) readDir(path string) (err error) {
	some, code, err := c.adapter.Propfind(c.opt.toAbsPath(path), "infinity")
	items := some.Propfinds
	if code == 404 {
		err = nil
		items = []Propfind{}
	}
	if err != nil {
		return
	}
	for _, item := range items {
		relPath := c.opt.toRelPath(item.GetNormalizedAbsPath())
		if _, exists := c.items[relPath]; exists {
			continue
		}
		c.itemPaths = append(c.itemPaths, relPath)
		c.items[relPath] = item
		if item.IsCollection() && relPath != path {
			defer c.readDir(relPath)
		}
	}
	return
}
