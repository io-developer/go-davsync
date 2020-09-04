package webdav

import (
	"fmt"
	"io"

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

func (c *Client) ReadTree() (parents map[string]client.Resource, children map[string]client.Resource, err error) {
	return c.tree.ReadTree()
}

func (c *Client) GetResource(path string) (res client.Resource, exists bool, err error) {
	return c.tree.GetResource(path)
}

func (c *Client) MakeDir(path string) error {
	return c.MakeDirAbs(c.opt.toAbsPath(path))
}

func (c *Client) MakeDirAbs(absPath string) error {
	absPath = util.PathNormalize(absPath, true)
	code, err := c.adapter.Mkcol(absPath)
	if err == nil && code >= 200 && code < 300 {
		return nil
	}
	return err
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
