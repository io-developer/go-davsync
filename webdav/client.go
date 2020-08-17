package webdav

import (
	"fmt"
	"io"

	"github.com/io-developer/davsync/model"
)

type Client struct {
	model.Client

	adapter *Adapter
}

func NewClient(adapter *Adapter) *Client {
	return &Client{
		adapter: adapter,
	}
}

func (c *Client) ReadTree() (paths []string, nodes map[string]model.Node, err error) {
	return c.adapter.ReadTree()
}

func (c *Client) AddDir(path string, recursive bool) error {
	var code int
	var err error
	if recursive {
		code, err = c.adapter.MkcolRecursive(path)
	} else {
		code, err = c.adapter.Mkcol(path)
	}
	if err != nil {
		return err
	}
	if code == 201 {
		return nil
	}
	return fmt.Errorf("Webdav AddDir (MKCOL) code: %d", code)
}

func (c *Client) AddFile(path string, content io.Reader) error {
	code, err := c.adapter.PutFile(path, content)
	if err != nil {
		return err
	}
	if code == 201 {
		return nil
	}
	return fmt.Errorf("Webdav AddFile (PUT) code: %d", code)
}
