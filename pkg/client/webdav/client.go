package webdav

import (
	"fmt"
	"io"
	"log"
	"regexp"
	"strings"

	"github.com/io-developer/go-davsync/pkg/client"
)

type Client struct {
	client.Client

	adapter       *Adapter
	propfinds     map[string]Propfind
	propfindPaths []string
}

func NewClient(adapter *Adapter) *Client {
	return &Client{
		adapter: adapter,
	}
}

func (c *Client) ReadTree() (paths []string, resources map[string]client.Resource, err error) {
	_, err = c.GetPropfinds()
	if err != nil {
		return
	}
	paths = c.propfindPaths
	resources = map[string]client.Resource{}
	for path, propfind := range c.propfinds {
		resources[path] = client.Resource{
			Path:     path,
			AbsPath:  propfind.GetHrefUnicode(),
			Name:     propfind.DisplayName,
			IsDir:    propfind.IsCollection(),
			Size:     propfind.ContentLength,
			UserData: propfind,
		}
	}
	log.Printf("DAV ReadTree:\n%#v\n\n", resources)
	return
}

func (c *Client) GetPropfinds() (map[string]Propfind, error) {
	if c.propfinds != nil {
		return c.propfinds, nil
	}
	c.propfindPaths = []string{}
	c.propfinds = map[string]Propfind{}
	err := c.ReadPropfinds("/", &c.propfindPaths, c.propfinds)
	return c.propfinds, err
}

func (c *Client) ReadPropfinds(
	path string,
	outPaths *[]string,
	outPropfinds map[string]Propfind,
) (err error) {
	some, err := c.adapter.Propfind(path, "infinity")
	if err != nil {
		return
	}
	for _, item := range some.Propfinds {
		absPath := item.GetHrefUnicode()
		relPath := strings.TrimPrefix(absPath, c.adapter.BasePath)
		if _, exists := outPropfinds[relPath]; exists {
			continue
		}
		*outPaths = append(*outPaths, relPath)
		outPropfinds[relPath] = item
		if item.IsCollection() && relPath != path {
			defer c.ReadPropfinds(relPath, outPaths, outPropfinds)
		}
	}
	return
}

func (c *Client) MakeDir(path string, recursive bool) error {
	if recursive {
		return c.makeDirRecursive(path)
	}
	_, err := c.adapter.Mkcol(path)
	return err
}

func (c *Client) MakeDirFor(filePath string) error {
	re, err := regexp.Compile("(^|/+)[^/]+$")
	if err != nil {
		return err
	}
	dir := re.ReplaceAllString(filePath, "")
	return c.makeDirRecursive(dir)
}

func (c *Client) makeDirRecursive(path string) error {
	parts := strings.Split(strings.Trim(path, "/"), "/")
	total := len(parts)
	if total < 1 {
		return nil
	}
	dir := ""
	for _, part := range parts {
		dir += "/" + part
		code, err := c.adapter.Mkcol(dir)
		if err != nil && code != 409 {
			return err
		}
	}
	return nil
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
