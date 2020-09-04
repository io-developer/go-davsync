package webdav

import (
	"fmt"
	"io"
	"regexp"
	"strings"

	"github.com/io-developer/go-davsync/pkg/client"
	"github.com/io-developer/go-davsync/pkg/util"
)

type Client struct {
	client.Client

	opt         Options
	adapter     *Adapter
	tree        client.Tree
	createdDirs map[string]string
}

func NewClient(opt Options) *Client {
	return &Client{
		opt:         opt,
		adapter:     NewAdapter(opt),
		tree:        NewTree(opt),
		createdDirs: make(map[string]string),
	}
}

func (c *Client) SetTree(t client.Tree) error {
	if t == nil {
		return fmt.Errorf("Unexpected non-nil Tree")
	}
	c.tree = t
	return nil
}

func (c *Client) ToAbsPath(relPath string) string {
	return c.opt.toAbsPath(relPath)
}

func (c *Client) ToRelativePath(absPath string) string {
	return c.opt.toRelPath(absPath)
}

func (c *Client) ReadTreeParents() (absPaths []string, items map[string]client.Resource, err error) {
	return c.tree.ReadParents()
}

func (c *Client) ReadTree() (parents map[string]client.Resource, children map[string]client.Resource, err error) {
	return c.tree.ReadTree()
}

func (c *Client) GetResource(path string) (res client.Resource, exists bool, err error) {
	return c.tree.GetResource(path)
}

func (c *Client) MakeDir(path string, recursive bool) error {
	if recursive {
		return c.makeDirRecursive(c.opt.toAbsPath(path))
	}
	_, err := c.makeDir(c.opt.toAbsPath(path))
	return err
}

func (c *Client) MakeDirFor(filePath string) error {
	re, err := regexp.Compile("(^|/+)[^/]+$")
	if err != nil {
		return err
	}
	dir := re.ReplaceAllString(filePath, "")
	return c.makeDirRecursive(c.opt.toAbsPath(dir))
}

func (c *Client) makeDirRecursive(absPath string) error {
	parts := strings.Split(strings.Trim(absPath, "/"), "/")
	total := len(parts)
	if total < 1 {
		return nil
	}
	subDir := "/"
	for _, part := range parts {
		if part != "" {
			subDir += part + "/"
			code, err := c.makeDir(subDir)
			if err != nil && code != 409 {
				return err
			}
		}
	}
	return nil
}

func (c *Client) makeDir(absPath string) (code int, err error) {
	absPath = util.PathNormalize(absPath, true)
	_, parents, err := c.tree.ReadParents()
	if _, exists := parents[absPath]; exists {
		return 200, nil
	}
	if _, exists := c.createdDirs[absPath]; exists {
		return 200, nil
	}
	path := c.opt.toRelPath(absPath)
	_, items, err := c.tree.ReadTree()
	if err != nil {
		return 0, err
	}
	if item, exists := items[path]; exists && item.IsDir {
		return 200, nil
	}
	code, err = c.adapter.Mkcol(absPath)
	if err == nil && code >= 200 && code < 300 {
		c.createdDirs[absPath] = absPath
	}
	return
}

func (c *Client) ReadFile(path string) (reader io.ReadCloser, err error) {
	reader, code, err := c.adapter.GetFile(c.opt.toAbsPath(path))
	if err != nil {
		return
	}
	if code == 200 {
		return
	}
	err = fmt.Errorf("Webdav ReadFile (GET) code: %d", code)
	return
}

func (c *Client) WriteFile(path string, content io.ReadCloser, size int64) error {
	code, err := c.adapter.PutFile(c.opt.toAbsPath(path), content, size)
	if err != nil {
		return err
	}
	if code == 201 {
		return nil
	}
	return fmt.Errorf("Webdav WriteFile (PUT) code: %d", code)
}

func (c *Client) MoveFile(srcPath, dstPath string) error {
	code, err := c.adapter.MoveFile(
		c.opt.toAbsPath(srcPath),
		c.opt.toAbsPath(dstPath),
	)
	if err != nil {
		return err
	}
	if code >= 200 && code < 300 {
		return nil
	}
	return fmt.Errorf("Webdav MoveFile (MOVE) code: %d", code)
}

func (c *Client) DeleteFile(path string) error {
	code, err := c.adapter.DeleteFile(c.opt.toAbsPath(path))
	if err != nil {
		return err
	}
	if code >= 200 && code < 300 {
		return nil
	}
	return fmt.Errorf("Webdav DeleteFile (DELETE) code: %d", code)
}
