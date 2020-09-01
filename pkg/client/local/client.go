package local

import (
	"io"
	"os"
	"path/filepath"

	"github.com/io-developer/go-davsync/pkg/client"
	"github.com/io-developer/go-davsync/pkg/util"
)

type Client struct {
	client.Client

	opt Options
}

func NewClient(opt Options) *Client {
	return &Client{
		opt: opt,
	}
}

func (c *Client) ReadParents() (absPaths []string, nodes map[string]client.Resource, err error) {
	return
}

func (c *Client) ReadTree() (paths []string, nodes map[string]client.Resource, err error) {
	paths = []string{}
	nodes = map[string]client.Resource{}
	err = filepath.Walk(c.opt.BaseDir, func(absPath string, info os.FileInfo, err error) error {
		res := c.toResource(absPath, info)
		path := res.Path
		paths = append(paths, path)
		nodes[path] = res
		return nil
	})
	return
}

func (c *Client) GetResource(path string) (res client.Resource, exists bool, err error) {
	absPath := c.opt.toAbsPath(path)
	info, err := os.Stat(absPath)
	if err == nil {
		res = c.toResource(absPath, info)
		exists = true
		return
	}
	if err == os.ErrNotExist {
		err = nil
	}
	return
}

func (c *Client) MakeDir(path string, recursive bool) error {
	absPath := c.opt.toAbsPath(path)
	if recursive {
		return os.MkdirAll(absPath, c.opt.DirMode)
	}
	return os.Mkdir(absPath, c.opt.DirMode)
}

func (c *Client) ReadFile(path string) (reader io.ReadCloser, err error) {
	return os.Open(c.opt.toAbsPath(path))
}

func (c *Client) WriteFile(path string, content io.ReadCloser, size int64) error {
	absPath := c.opt.toAbsPath(path)
	file, err := os.OpenFile(absPath, os.O_CREATE|os.O_WRONLY, c.opt.FileMode)
	if err != nil {
		return err
	}
	_, err = io.Copy(file, content)
	if err != nil {
		return err
	}
	return content.Close()
}

func (c *Client) MoveFile(srcPath, dstPath string) error {
	return os.Rename(c.opt.toAbsPath(srcPath), c.opt.toAbsPath(dstPath))
}

func (c *Client) toResource(absPath string, info os.FileInfo) client.Resource {
	absPath = util.PathNormalize(absPath, info.IsDir())
	return client.Resource{
		AbsPath:  absPath,
		Path:     c.opt.toRelPath(absPath),
		Name:     info.Name(),
		IsDir:    info.IsDir(),
		Size:     info.Size(),
		ModTime:  info.ModTime(),
		UserData: info,
	}
}
