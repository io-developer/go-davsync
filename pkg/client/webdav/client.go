package webdav

import (
	"fmt"
	"io"

	"github.com/io-developer/go-davsync/pkg/client"
)

type Client struct {
	client.Client

	adapter *Adapter
}

func NewClient(adapter *Adapter) *Client {
	return &Client{
		adapter: adapter,
	}
}

func (c *Client) ReadTree() (paths []string, nodes map[string]client.Resource, err error) {
	return c.adapter.ReadTree()
}

func (c *Client) MakeDir(path string, recursive bool) error {
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
	return fmt.Errorf("Webdav MakeDir (MKCOL) code: %d", code)
}

func (c *Client) ReadFile(path string) (reader io.ReadCloser, err error) {
	reader, code, err := c.adapter.GetFile(path)
	if err != nil {
		return
	}
	if code == 200 {
		return
	}
	err = fmt.Errorf("Webdav ReadFile (GET) code: %d", code)
	return
}

func (c *Client) WriteFile(path string, content io.ReadCloser) error {
	code, err := c.adapter.PutFile(path, content)
	if err != nil {
		return err
	}
	if code == 201 {
		return nil
	}
	return fmt.Errorf("Webdav WriteFile (PUT) code: %d", code)
}
