package webdav

import (
	"fmt"
	"log"
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
		reader := newFileTreeReader(c.opt, 4)
		err = reader.ReadDir("/")
		c.itemPaths, c.items = reader.parsedPaths, reader.parsedItems
	}
	return c.itemPaths, c.items, err
}

type fileTreeReader struct {
	opt         Options
	numThreads  int
	parsedItems map[string]Propfind
	parsedPaths []string
}

func newFileTreeReader(opt Options, numThreads int) *fileTreeReader {
	if numThreads < 1 {
		numThreads = 1
	}
	return &fileTreeReader{
		opt:        opt,
		numThreads: numThreads,
	}
}

func (r *fileTreeReader) log(msg string) {
	log.Printf("Dav tree: %s\n", msg)
}

func (r *fileTreeReader) ReadDir(path string) (err error) {
	queueCounter := 0
	logMain := func(msg string) {
		r.log(fmt.Sprintf("[main, queue=%d]: %s", queueCounter, msg))
	}

	paths := []string{}
	items := map[string]Propfind{}

	queue := make(chan treeMsg)
	parsed := make(chan treeMsg)
	completed := make(chan treeMsg)
	errors := make(chan treeMsg)

	numThreads := r.numThreads
	for i := 0; i < numThreads; i++ {
		go r.thread(i, queue, parsed, completed, errors)
	}

	queueCounter++
	go func() {
		queue <- treeMsg{
			relPath: path,
			depth:   "infinity",
		}
	}()

	logMain("Starting..")
	inProgress := true
	for inProgress {
		select {
		case msg, success := <-parsed:
			if !success {
				inProgress = false
				break
			}
			if _, exists := items[msg.relPath]; !exists {
				logMain(fmt.Sprintf("Parsed new: %s", msg.relPath))

				paths = append(paths, msg.relPath)
				items[msg.relPath] = msg.payload

				if msg.payload.IsCollection() && msg.relPath != path {
					logMain("  subdir found, pushing to queue")
					queueCounter++
					go func(msg treeMsg) {
						queue <- msg
					}(treeMsg{
						relPath: msg.relPath,
						depth:   msg.depth,
					})
				}
			} else {
				logMain(fmt.Sprintf("Parsed existed: %s", msg.relPath))
			}

		case msg, success := <-completed:
			queueCounter--
			logMain(fmt.Sprintf("Complete: %s", msg.relPath))
			if !success {
				inProgress = false
				break
			}

		case msg, _ := <-errors:
			queueCounter--
			logMain(fmt.Sprintf("ERROR: %s", msg.relPath))
			err = msg.err
			inProgress = false
			break
		}

		inProgress = inProgress && queueCounter > 0
	}

	if err == nil {
		logMain("Complete")
		r.parsedPaths = paths
		r.parsedItems = items
	} else {
		logMain(fmt.Sprintf("Stopped with error: %v", err))
	}

	close(queue)
	close(parsed)
	close(completed)
	close(errors)

	return
}

func (r *fileTreeReader) thread(id int, queue, parsed, completed, errors chan treeMsg) {
	curPath := "-"
	logThread := func(msg string) {
		r.log(fmt.Sprintf("[thread %d] '%s': %s", id, curPath, msg))
	}
	logThread("Thread started..")

	adapter := NewAdapter(r.opt)
	for {
		select {
		case msg, success := <-queue:
			if !success {
				logThread("Thread exited")
				return
			}
			curPath = msg.relPath

			logThread("propfind..")
			some, code, err := adapter.Propfind(r.opt.toAbsPath(msg.relPath), "infinity")
			items := some.Propfinds
			if code == 404 {
				logThread("http 404")
				err = nil
				items = []Propfind{}
			}
			if err != nil {
				logThread(fmt.Sprintf("ERROR code=%d, err: %v", code, err))

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
			curPath = "-"
		}
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
