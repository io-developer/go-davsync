package yadiskrest

import (
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"path/filepath"
	"strings"
	"time"

	"github.com/io-developer/davsync/model"
)

type ClientOpt struct {
	ApiUri    string
	AuthToken string
}

type Client struct {
	//	model.Client

	BaseDir    string
	RetryLimit int
	RetryDelay time.Duration

	opt         ClientOpt
	httpClient  http.Client
	baseHeaders map[string]string
	resources   *Resources
}

func NewClient(opt ClientOpt) *Client {
	return &Client{
		RetryLimit: 3,
		RetryDelay: 1 * time.Second,

		opt:        opt,
		httpClient: http.Client{},
		baseHeaders: map[string]string{
			"Connection": "keep-alive",
		},
		resources: nil,
	}
}

func (c *Client) ReadTree() (paths []string, nodes map[string]model.Node, err error) {
	paths = []string{}
	nodes = map[string]model.Node{}
	r, err := c.GetResources()
	if err != nil {
		return
	}
	for _, item := range r.Items {
		path := c.relPathFrom(item.Path)
		nodes[path] = model.Node{
			Path:     path,
			AbsPath:  item.Path,
			IsDir:    item.IsDir(),
			Name:     item.Name,
			Size:     item.Size,
			UserData: item,
		}
	}
	return
}

/*
func (c *Client) MakeDir(path string, recursive bool) error {

}

func (c *Client) ReadFile(path string) (reader io.ReadCloser, err error) {

}

func (c *Client) WriteFile(path string, content io.ReadCloser) error {

}
*/

func (c *Client) GetResources() (r Resources, err error) {
	if c.resources != nil {
		return *c.resources, nil
	}
	bytes, err := c.requestBytes("GET", "/resources/files", url.Values{
		"limit": []string{"999999"},
	})
	if err != nil {
		return
	}
	r, err = ResourcesParse(bytes)
	if err == nil {
		c.resources = &r
	}
	return
}

func (c *Client) relPathFrom(absPath string) string {
	prefix := "disk:" + strings.TrimRight(c.BaseDir, "/")
	relpath := strings.TrimPrefix(absPath, prefix)
	return filepath.Join("/", relpath)
}

// http impl

func (c *Client) requestBytes(method, path string, query url.Values) ([]byte, error) {
	resp, err := c.request(method, path, query)
	if err != nil {
		return nil, err
	}
	return ioutil.ReadAll(resp.Body)
}

func (c *Client) request(method, path string, query url.Values) (resp *http.Response, err error) {
	req, err := c.createRequest("GET", path, query, nil)
	if err != nil {
		return
	}
	return c.sendRequest(req)
}

func (c *Client) sendRequest(req *http.Request) (resp *http.Response, err error) {
	for i := 0; i < c.RetryLimit; i++ {
		resp, err = c.httpClient.Do(req)
		if err != nil {
			continue
		}
		if resp.StatusCode == 429 {
			time.Sleep(c.RetryDelay)
			continue
		}
		break
	}
	return
}

func (c *Client) newReq(method, path string, query url.Values) (*http.Request, error) {
	return c.createRequest("GET", path, query, nil)
}

func (c *Client) createRequest(method, path string, query url.Values, body io.Reader) (*http.Request, error) {
	uri := c.buildURI(path, query)
	log.Printf("createRequest\n  path: %s\n  uri: %s\n  method: %s\n\n", path, uri, method)

	req, err := http.NewRequest(method, uri, body)
	if err != nil {
		return req, err
	}
	for key, val := range c.baseHeaders {
		req.Header.Add(key, val)
	}
	return c.auth(req), err
}

func (c *Client) auth(req *http.Request) *http.Request {
	req.Header.Add("Authorization", fmt.Sprintf("OAuth %s", c.opt.AuthToken))
	return req
}

func (c *Client) buildURI(path string, query url.Values) string {
	uri := fmt.Sprintf(
		"%s/%s",
		strings.TrimRight(c.opt.ApiUri, "/"),
		strings.TrimLeft(path, "/"),
	)
	q := query.Encode()
	if q != "" {
		uri += "?" + q
	}
	return uri
}
