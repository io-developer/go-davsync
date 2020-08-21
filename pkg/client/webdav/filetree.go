package webdav

import (
	"fmt"
	"strings"
	"time"

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
		c.itemPaths, c.items = reader.parsedPaths, reader.parsedItems
	}
	return c.itemPaths, c.items, err
}

type fileTreeReader struct {
	opt         Options
	parsedItems map[string]Propfind
	parsedPaths []string
}

func newFileTreeReader(opt Options) *fileTreeReader {
	return &fileTreeReader{
		opt: opt,
	}
}

func (r *fileTreeReader) ReadDir(path string) (err error) {
	fmt.Println("ReadDir ", path)

	paths := []string{}
	items := map[string]Propfind{}

	queue := make(chan treeMsg)
	parsed := make(chan treeMsg)
	completed := make(chan treeMsg)
	errors := make(chan treeMsg)

	numThreads := 1
	fmt.Println("  stating threads", numThreads)
	for i := 0; i < numThreads; i++ {
		go r.thread(i, queue, parsed, completed, errors)
	}
	fmt.Println("  pushing root path msg", path)
	go func() {
		time.Sleep(time.Second)
		queue <- treeMsg{
			relPath: path,
			depth:   "infinity",
		}
	}()
	fmt.Println("  starting main loop...")
	inProgress := true
	for inProgress {
		select {
		case msg, success := <-parsed:
			fmt.Printf("ReadDir parsed: %#v\n", msg)
			if !success {
				inProgress = false
				break
			}
			if _, exists := items[msg.relPath]; !exists {
				fmt.Println("  adding new payload ", msg.relPath)
				paths = append(paths, msg.relPath)
				items[msg.relPath] = msg.payload

				if msg.payload.IsCollection() && msg.relPath != path {
					fmt.Println("  reading subdir ", msg.relPath)
					go func(msg treeMsg) {
						queue <- msg
					}(treeMsg{
						relPath: msg.relPath,
						depth:   msg.depth,
					})
				}
			}

		case msg, success := <-completed:
			fmt.Printf("ReadDir completed: %#v\n", msg)
			if !success {
				inProgress = false
				break
			}

		case msg, _ := <-errors:
			fmt.Printf("ReadDir error: %#v\n", msg)
			err = msg.err
			inProgress = false
			break
		}
	}

	fmt.Println("ReadDir loop stopped, err:", err)
	if err == nil {
		r.parsedPaths = paths
		r.parsedItems = items
	}

	close(queue)
	close(parsed)
	close(completed)
	close(errors)

	return
}

func (r *fileTreeReader) thread(id int, queue, parsed, completed, errors chan treeMsg) {
	fmt.Println("Start thread ", id)
	adapter := NewAdapter(r.opt)
	for {
		select {
		case msg, success := <-queue:
			if !success {
				return
			}
			fmt.Println(id, "Tree read dir", msg.relPath)

			some, code, err := adapter.Propfind(r.opt.toAbsPath(msg.relPath), "infinity")
			items := some.Propfinds
			if code == 404 {
				err = nil
				items = []Propfind{}
			}
			if err != nil {
				fmt.Println(id, " thread error", code, err)

				msg.err = err
				msg.errHttpCode = code
				errors <- msg
				return
			}
			for _, item := range items {
				relPath := r.opt.toRelPath(item.GetNormalizedAbsPath())
				parsed <- treeMsg{
					payload: item,
					relPath: relPath,
					depth:   msg.depth,
				}
			}

			completed <- msg
		}
		fmt.Println(id, " thread loop")
	}
}

type treeMsg struct {
	hasPayload  bool
	payload     Propfind
	relPath     string
	depth       string
	err         error
	errHttpCode int
}
