package webdav

import (
	"fmt"
	"io"
	"log"
	"regexp"
	"strings"

	"github.com/io-developer/go-davsync/pkg/client"
)

type ClientOpt struct {
	BaseDir       string
	DavUri        string
	AuthToken     string
	AuthTokenType string
	AuthUser      string
	AuthPass      string
}

type Client struct {
	client.Client

	opt         Options
	adapter     *Adapter
	fileTree    *FileTree
	createdDirs map[string]string
}

func NewClient(opt Options) *Client {
	return &Client{
		opt:         opt,
		adapter:     NewAdapter(opt),
		fileTree:    NewFileTree(opt),
		createdDirs: make(map[string]string),
	}
}

func (c *Client) ReadTree() (paths []string, resources map[string]client.Resource, err error) {
	_, err = c.fileTree.GetParents()
	if err != nil {
		return
	}
	paths, propfinds, err := c.fileTree.GetItems()
	if err != nil {
		return
	}
	resources = map[string]client.Resource{}
	for path, propfind := range propfinds {
		resources[path] = client.Resource{
			Path:     path,
			AbsPath:  propfind.GetNormalizedAbsPath(),
			Name:     propfind.DisplayName,
			IsDir:    propfind.IsCollection(),
			Size:     propfind.ContentLength,
			UserData: propfind,
		}
	}
	//log.Printf("DAV ReadTree:\n%#v\n\n", resources)
	return
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
	log.Println("makeDir", absPath)

	absPath = client.PathNormalize(absPath, true)
	parents, err := c.fileTree.GetParents()
	if _, exists := parents[absPath]; exists {
		log.Println("  exists in parents")
		return 200, nil
	}
	if _, exists := c.createdDirs[absPath]; exists {
		log.Println("  exists in createdDirs")
		return 200, nil
	}
	path := c.opt.toRelPath(absPath)
	_, items, err := c.fileTree.GetItems()
	if err != nil {
		return 0, err
	}
	if item, exists := items[path]; exists && item.IsCollection() {
		log.Println("  exists in items")
		return 200, nil
	}
	code, err = c.adapter.Mkcol(absPath)
	if err == nil && code >= 200 && code < 300 {
		log.Println("  DIR CREATED", absPath)
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

func (c *Client) WriteFile(path string, content io.ReadCloser) error {
	code, err := c.adapter.PutFile(c.opt.toAbsPath(path), content)
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
