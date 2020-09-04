package yadisk

import (
	"io"

	"github.com/io-developer/go-davsync/pkg/client"
	"github.com/io-developer/go-davsync/pkg/client/webdav"
	"github.com/io-developer/go-davsync/pkg/client/yadiskrest"
)

type Client struct {
	client.Client

	dav  *webdav.Client
	rest *yadiskrest.Client
}

func NewClient(dav *webdav.Client, rest *yadiskrest.Client) *Client {
	return &Client{
		dav:  dav,
		rest: rest,
	}
}

func (c *Client) ToAbsPath(relPath string) string {
	return c.dav.ToAbsPath(relPath)
}

func (c *Client) ToRelativePath(absPath string) string {
	return c.dav.ToRelativePath(absPath)
}

func (c *Client) ReadTree() (parents, children map[string]client.Resource, err error) {
	return c.rest.ReadTree()
}
func (c *Client) ReadResource(path string) (res client.Resource, exists bool, err error) {
	return c.rest.ReadResource(path)
}

func (c *Client) MakeDir(path string) error {
	return c.dav.MakeDir(path)
}
func (c *Client) MakeDirAbs(path string) error {
	return c.dav.MakeDirAbs(path)
}

func (c *Client) ReadFile(path string) (reader io.ReadCloser, err error) {
	return c.dav.ReadFile(path)
}
func (c *Client) WriteFile(path string, content io.ReadCloser, size int64) error {
	return c.dav.WriteFile(path, content, size)
}
func (c *Client) MoveFile(srcPath, dstPath string) error {
	return c.dav.MoveFile(srcPath, dstPath)
}
func (c *Client) DeleteFile(path string) error {
	return c.dav.DeleteFile(path)
}
