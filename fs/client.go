package fs

import (
	"bufio"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/io-developer/davsync/model"
)

type Client struct {
	model.Client

	BaseDir  string
	DirMode  os.FileMode
	FileMode os.FileMode
}

func NewClient(baseDir string) *Client {
	return &Client{
		BaseDir:  baseDir,
		DirMode:  755,
		FileMode: 644,
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

func (c *Client) ReadFile(path string) (io.Reader, error) {
	realpath := filepath.Join(c.BaseDir, path)
	file, err := os.Open(realpath)
	if err != nil {
		return nil, err
	}
	return bufio.NewReader(file), err
}

func (c *Client) AddDir(path string, recursive bool) error {
	realpath := filepath.Join(c.BaseDir, path)
	if recursive {
		return os.MkdirAll(realpath, c.DirMode)
	}
	return os.Mkdir(realpath, c.DirMode)
}
