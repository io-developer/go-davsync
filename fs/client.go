package fs

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/io-developer/davsync/model"
)

type Client struct {
	BaseDir string
}

func NewClient(baseDir string) *Client {
	return &Client{
		BaseDir: baseDir,
	}
}

func (c *Client) ReadTree() (paths []string, nodes map[string]model.Node, err error) {
	paths = []string{}
	nodes = map[string]model.Node{}
	err = filepath.Walk(c.BaseDir, func(absPath string, info os.FileInfo, err error) error {
		if info.IsDir() && !strings.HasSuffix(absPath, "/") {
			absPath += "/"
		}
		path := strings.TrimPrefix(absPath, strings.TrimRight(c.BaseDir, "/"))
		paths = append(paths, path)
		nodes[path] = model.Node{
			AbsPath:  absPath,
			Path:     path,
			Name:     info.Name(),
			IsDir:    info.IsDir(),
			FileInfo: &info,
		}
		return nil
	})
	return
}
