package webdav

import (
	"fmt"

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

func (c *Client) Mkdir(path string, recursive bool) error {
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
	return fmt.Errorf("WebDAV MKCOL code: %d", code)
}
