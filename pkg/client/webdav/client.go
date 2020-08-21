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

	opt             Options
	adapter         *Adapter
	parentPropfinds map[string]Propfind
	propfinds       map[string]Propfind
	propfindPaths   []string
	createdDirs     map[string]string
}

func NewClient(opt Options) *Client {
	return &Client{
		opt:         opt,
		adapter:     NewAdapter(opt),
		createdDirs: make(map[string]string),
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
			AbsPath:  propfind.GetNormalizedAbsPath(),
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
	var err error
	c.propfindPaths = []string{}
	c.propfinds = map[string]Propfind{}
	c.parentPropfinds, err = c.readParents()
	if err == nil {
		err = c.ReadPropfinds("/", &c.propfindPaths, c.propfinds)
	}
	return c.propfinds, err
}

func (c *Client) readParents() (parents map[string]Propfind, err error) {
	parents = make(map[string]Propfind)
	parts := strings.Split(strings.Trim(c.opt.BaseDir, "/"), "/")
	total := len(parts)
	if total < 1 {
		return
	}
	path := ""
	for _, part := range parts {
		path += "/" + part
		some, code, perr := c.adapter.Propfind(path, "0")
		err = perr
		if code == 404 {
			err = nil
			return
		}
		if err != nil {
			return
		}
		if len(some.Propfinds) < 1 {
			return
		}
		normPath := client.PathNormalize(path, true)
		parents[normPath] = some.Propfinds[0]
	}
	return
}

func (c *Client) ReadPropfinds(
	path string,
	outPaths *[]string,
	outPropfinds map[string]Propfind,
) (err error) {
	some, code, err := c.adapter.Propfind(c.opt.toAbsPath(path), "infinity")
	items := some.Propfinds
	if code == 404 {
		err = nil
		items = []Propfind{}
	}
	if err != nil {
		return
	}
	for _, item := range items {
		relPath := c.opt.toRelPath(item.GetNormalizedAbsPath())
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
	if _, exists := c.parentPropfinds[absPath]; exists {
		log.Println("  exists in parents")
		return 200, nil
	}
	if _, exists := c.createdDirs[absPath]; exists {
		log.Println("  exists in createdDirs")
		return 200, nil
	}
	path := c.opt.toRelPath(absPath)
	if propfind, exists := c.propfinds[path]; exists && propfind.IsCollection() {
		log.Println("  exists in propfinds")
		return 200, nil
	}
	code, err = c.adapter.Mkcol(absPath)
	if err == nil && code >= 200 && code < 300 {
		log.Println("  MADE")
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
