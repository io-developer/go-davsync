package fs

import (
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/io-developer/go-davsync/pkg/client"
)

type Client struct {
	client.Client

	BaseDir  string
	DirMode  os.FileMode
	FileMode os.FileMode
}

func NewClient(baseDir string) *Client {
	return &Client{
		BaseDir:  baseDir,
		DirMode:  0755,
		FileMode: 0644,
	}
}

func (c *Client) ReadTree() (paths []string, nodes map[string]client.Resource, err error) {
	paths = []string{}
	nodes = map[string]client.Resource{}
	err = filepath.Walk(c.BaseDir, func(absPath string, info os.FileInfo, err error) error {
		absPath = client.PathNormalize(absPath, info.IsDir())
		path := strings.TrimPrefix(absPath, strings.TrimRight(c.BaseDir, "/"))
		paths = append(paths, path)
		nodes[path] = client.Resource{
			AbsPath:  absPath,
			Path:     path,
			Name:     info.Name(),
			IsDir:    info.IsDir(),
			Size:     info.Size(),
			FileInfo: &info,
		}
		return nil
	})
	return
}

func (c *Client) MakeDir(path string, recursive bool) error {
	realpath := filepath.Join(c.BaseDir, path)
	if recursive {
		return os.MkdirAll(realpath, c.DirMode)
	}
	return os.Mkdir(realpath, c.DirMode)
}

func (c *Client) ReadFile(path string) (reader io.ReadCloser, err error) {
	realpath := filepath.Join(c.BaseDir, path)
	return os.Open(realpath)
}

func (c *Client) WriteFile(path string, content io.ReadCloser) error {
	realpath := filepath.Join(c.BaseDir, path)
	file, err := os.OpenFile(realpath, os.O_CREATE|os.O_WRONLY, c.FileMode)
	if err != nil {
		return err
	}
	_, err = io.Copy(file, content)
	if err != nil {
		return err
	}
	return content.Close()
}
