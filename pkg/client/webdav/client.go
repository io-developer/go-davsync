package webdav

import (
	"fmt"
	"io"

	"github.com/io-developer/go-davsync/pkg/client"
	"github.com/io-developer/go-davsync/pkg/util"
)

type Client struct {
	client.Client

	opt     Options
	adapter *Adapter
}

func NewClient(opt Options) *Client {
	return &Client{
		opt:     opt,
		adapter: NewAdapter(opt),
	}
}

func (c *Client) ToAbsPath(relPath string) string {
	return c.opt.toAbsPath(relPath)
}

func (c *Client) ToRelativePath(absPath string) string {
	return c.opt.toRelPath(absPath)
}

func (c *Client) ReadTree() (parents map[string]client.Resource, children map[string]client.Resource, err error) {
	reader := newTreeReader(c.opt, 4)
	parentItems, err := reader.readParents()
	if err != nil {
		return
	}
	parents = map[string]client.Resource{}
	for path, propfind := range parentItems {
		parents[path] = propfind.ToResource(path)
	}
	err = reader.ReadDir("/")
	if err != nil {
		return
	}
	children = map[string]client.Resource{}
	for path, propfind := range reader.parsedItems {
		children[path] = propfind.ToResource(path)
	}
	return
}

func (c *Client) ReadResource(path string) (res client.Resource, exists bool, err error) {
	some, code, err := c.adapter.Propfind(c.opt.toAbsPath(path), "0")
	if err == nil && len(some.Propfinds) == 1 {
		propfind := some.Propfinds[0]
		res = propfind.ToResource(path)
		exists = true
		return
	}
	if code == 404 {
		err = nil
	}
	return
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
