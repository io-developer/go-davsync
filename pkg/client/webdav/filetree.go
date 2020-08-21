package webdav

import (
	"fmt"
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
		reader := newFileTreeReader(c.opt)
		err = reader.ReadDir("/")
		c.itemPaths, c.items = reader.itemPaths, reader.items
	}
	return c.itemPaths, c.items, err
}

type fileTreeReader struct {
	opt       Options
	items     map[string]Propfind
	itemPaths []string
}

func newFileTreeReader(opt Options) *fileTreeReader {
	return &fileTreeReader{
		opt:       opt,
		items:     map[string]Propfind{},
		itemPaths: []string{},
	}
}

func (r *fileTreeReader) ReadDir(path string) error {
	fmt.Println("ReadDir", path)

	queue := 0
	readCh := make(chan treeMsg)
	completeCh := make(chan treeMsg)
	errorCh := make(chan treeMsg)

	go func() {
		readCh <- treeMsg{
			relPath: path,
			depth:   "infinity",
		}
	}()
	for {
		select {
		case msg, success := <-readCh:
			if success {
				queue++
				fmt.Printf("Read dir requested (queue %d): %#v\n", queue, msg)
				go r.readDir(msg, readCh, completeCh, errorCh)
			} else {
				return nil
			}
		case msg, success := <-completeCh:
			if success {
				queue--
				fmt.Printf("Read dir complete (queue %d): %#v\n", queue, msg)
			}
			if !success || queue <= 0 {
				return nil
			}
		case msg, success := <-errorCh:
			if success {
				queue--
				fmt.Printf("Read dir error (queue %d): %#v\n", queue, msg)
				return msg.err
			}
			return nil
		}
	}
}

func (r *fileTreeReader) getAdapter() *Adapter {
	return NewAdapter(r.opt)
}

func (r *fileTreeReader) readDir(msg treeMsg, readCh, completeCh, errCh chan treeMsg) {
	fmt.Println("Tree read dir", msg.relPath)

	adapter := r.getAdapter()
	some, code, err := adapter.Propfind(r.opt.toAbsPath(msg.relPath), "infinity")
	items := some.Propfinds
	if code == 404 {
		err = nil
		items = []Propfind{}
	}
	if err != nil {
		msg.err = err
		errCh <- msg
		return
	}
	for _, item := range items {
		// TODO: fix concurrent map access!!
		relPath := r.opt.toRelPath(item.GetNormalizedAbsPath())
		if _, exists := r.items[relPath]; exists {
			continue
		}
		r.itemPaths = append(r.itemPaths, relPath)
		r.items[relPath] = item
		if item.IsCollection() && relPath != msg.relPath {
			readCh <- treeMsg{
				relPath: relPath,
				depth:   msg.depth,
			}
		}
	}
	completeCh <- msg
}

type treeMsg struct {
	relPath string
	depth   string
	err     error
}
