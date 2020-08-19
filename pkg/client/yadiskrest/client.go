package yadiskrest

import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/io-developer/go-davsync/pkg/client"
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

	opt           ClientOpt
	httpClient    http.Client
	baseHeaders   map[string]string
	resources     map[string]Resource
	resourcePaths []string
}

func NewClient(opt ClientOpt) *Client {
	return &Client{
		RetryLimit: 3,
		RetryDelay: 1 * time.Second,

		opt:        opt,
		httpClient: http.Client{},
		baseHeaders: map[string]string{
			"Accept":     "*/*",
			"Connection": "keep-alive",
		},
	}
}

func (c *Client) ReadTree() (paths []string, nodes map[string]client.Node, err error) {
	items, err := c.GetResources()
	if err != nil {
		return
	}
	nodes = map[string]client.Node{}
	for path, item := range items {
		nodes[path] = client.Node{
			Path:     path,
			AbsPath:  item.Path,
			IsDir:    item.IsDir(),
			Name:     item.Name,
			Size:     item.Size,
			UserData: item,
		}
	}
	return c.resourcePaths, nodes, nil
}

func (c *Client) MakeDir(path string, recursive bool) error {
	if recursive {
		return c.makeDirRecursive(path)
	}
	_, err := c.makeDir(path)
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
		code, err := c.makeDir(dir)
		if err != nil && code != 409 {
			return err
		}
	}
	return nil
}

func (c *Client) makeDir(path string) (code int, err error) {
	absPath := filepath.Join("/", c.BaseDir, path)
	req, err := c.createRequest("PUT", "/resources", url.Values{
		"path": []string{absPath},
	}, nil)
	if err != nil {
		return
	}
	resp, err := c.sendRequest(req)
	if err != nil {
		return
	}
	code = resp.StatusCode
	if code != 201 {
		err = fmt.Errorf("Expected mkdir code 201, got %d '%s'", code, resp.Status)
	}
	return
}

func (c *Client) ReadFile(path string) (reader io.ReadCloser, err error) {
	items, err := c.GetResources()
	if err != nil {
		return
	}
	item, exists := items[path]
	if !exists {
		err = fmt.Errorf("Resource not found '%s'", path)
		return
	}
	if !item.IsFile() {
		err = fmt.Errorf("Resource is not a file (%s) at '%s'", item.Type, path)
		return
	}
	if item.File == "" {
		err = fmt.Errorf("Resource download uri is empty at '%s'", path)
		return
	}
	req, err := http.NewRequest("GET", item.File, nil)
	if err != nil {
		return
	}
	resp, err := c.sendRequest(req)
	if err != nil {
		return
	}
	log.Println("item md5", item.Md5)
	return resp.Body, nil
}

func (c *Client) WriteFile(path string, content io.ReadCloser) error {
	absPath := filepath.Join("/", c.BaseDir, path)
	resp, err := c.request("GET", "/resources/upload", url.Values{
		"path":      []string{absPath},
		"overwrite": []string{"true"},
	})
	if err != nil {
		return err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf(
			"Unexpected response code %d '%s'",
			resp.StatusCode,
			resp.Status,
		)
	}
	bytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	info := &UploadInfo{}
	err = json.Unmarshal(bytes, &info)
	if err != nil {
		return err
	}
	log.Printf("UploadInfo:\n%#v\n", info)
	if info.Templated {
		return fmt.Errorf("Unexpected templated=true.\n  Info: %#v", info)
	}
	req, err := http.NewRequest(info.Method, info.Href, content)
	if err != nil {
		return err
	}
	resp, err = c.sendRequest(req)
	if err != nil {
		return err
	}
	if resp.StatusCode == 201 || resp.StatusCode == 202 {
		return nil
	}
	return fmt.Errorf("Upload failed with code %d, '%s'", resp.StatusCode, resp.Status)
}

func (c *Client) GetResources() (map[string]Resource, error) {
	if c.resources != nil {
		return c.resources, nil
	}
	bytes, err := c.requestBytes("GET", "/resources/files", url.Values{
		"limit": []string{"999999"},
	})
	if err != nil {
		return c.resources, err
	}
	r := &Resources{}
	err = json.Unmarshal(bytes, &r)
	if err != nil {
		return c.resources, err
	}
	c.resources = map[string]Resource{}
	c.resourcePaths = []string{}
	for _, item := range r.Items {
		path, isSubset := c.relPathFrom(item.Path)
		if isSubset {
			c.resources[path] = item
			c.resourcePaths = append(c.resourcePaths, path)
		}
	}
	return c.resources, nil
}

func (c *Client) relPathFrom(absPath string) (path string, isSubset bool) {
	prefix := "disk:" + strings.TrimRight(c.BaseDir, "/")
	path = filepath.Join("/", strings.TrimPrefix(absPath, prefix))
	isSubset = strings.HasPrefix(absPath, prefix)
	return
}

// http impl

func (c *Client) requestBytes(method, path string, query url.Values) ([]byte, error) {
	resp, err := c.request(method, path, query)
	if err != nil {
		return nil, err
	}
	log.Println("requestBytes code", resp.StatusCode, resp.Status)
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
