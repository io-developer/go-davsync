package yadiskrest

import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"time"

	"github.com/io-developer/go-davsync/pkg/client"
	"github.com/io-developer/go-davsync/pkg/log"
	"github.com/io-developer/go-davsync/pkg/util"
)

type Client struct {
	//	model.Client

	BaseDir    string
	RetryLimit int
	RetryDelay time.Duration

	opt         Options
	httpClient  http.Client
	baseHeaders map[string]string

	treeParents     map[string]Resource
	treeParentPaths []string
	treeItems       map[string]Resource
	treeItemPaths   []string
}

func NewClient(opt Options) *Client {
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

func (c *Client) ToAbsPath(relPath string) string {
	return c.opt.toAbsPath(relPath)
}

func (c *Client) ToRelativePath(absPath string) string {
	return c.opt.toRelPath(absPath)
}

func (c *Client) ReadTree() (parents map[string]client.Resource, children map[string]client.Resource, err error) {
	err = c.readTree()
	if err != nil {
		return
	}
	parents = map[string]client.Resource{}
	for absPath, parent := range c.treeParents {
		parents[absPath] = parent.ToResource(absPath)
	}
	children = map[string]client.Resource{}
	for path, item := range c.treeItems {
		children[path] = item.ToResource(path)
	}
	return
}

func (c *Client) ReadResource(path string) (res client.Resource, exists bool, err error) {
	resp, err := c.request("GET", "/resources/", url.Values{
		"path": []string{c.opt.toAbsPath(path)},
	})
	if resp.StatusCode == 404 {
		err = nil
		return
	}
	if err != nil {
		return
	}
	bytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return
	}
	yaRes := Resource{}
	err = json.Unmarshal(bytes, &yaRes)
	if err != nil {
		return
	}
	res = yaRes.ToResource(path)
	exists = true
	return
}

func (c *Client) MakeDir(path string) error {
	return c.MakeDirAbs(c.opt.toAbsPath(path))
}

func (c *Client) MakeDirAbs(absPath string) error {
	req, err := c.createRequest("PUT", "/resources", url.Values{
		"path": []string{absPath},
	}, nil)
	if err != nil {
		return err
	}
	resp, err := c.sendRequest(req)
	if err != nil {
		return err
	}
	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		return nil
	}
	return fmt.Errorf("Unexpected mkdir code %d '%s'", resp.StatusCode, resp.Status)
}

func (c *Client) ReadFile(path string) (reader io.ReadCloser, err error) {
	err = c.readTree()
	if err != nil {
		return
	}
	item, exists := c.treeItems[path]
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
	log.Debug("item md5", item.Md5)
	return resp.Body, nil
}

func (c *Client) WriteFile(path string, content io.ReadCloser, size int64) error {
	resp, err := c.request("GET", "/resources/upload", url.Values{
		"path":      []string{c.opt.toAbsPath(path)},
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
	log.Debug("UploadInfo:\n%#v\n", info)
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

func (c *Client) MoveFile(srcPath, dstPath string) error {
	req, err := c.createRequest("POST", "/resources/move", url.Values{
		"from":      []string{c.opt.toAbsPath(srcPath)},
		"path":      []string{c.opt.toAbsPath(dstPath)},
		"overwrite": []string{"true"},
	}, nil)
	if err != nil {
		return err
	}
	resp, err := c.sendRequest(req)
	if err != nil {
		return err
	}
	code := resp.StatusCode
	if code >= 200 && code < 300 {
		return nil
	}
	return fmt.Errorf("Unexpected MoveFile (MOVE) code: %d", code)
}

func (c *Client) DeleteFile(path string) error {
	permanently := "false"
	if c.opt.DeletePermanent {
		permanently = "true"
	}
	req, err := c.createRequest("DELETE", "/resources", url.Values{
		"path":        []string{c.opt.toAbsPath(path)},
		"permanently": []string{permanently},
	}, nil)
	if err != nil {
		return err
	}
	resp, err := c.sendRequest(req)
	if err != nil {
		return err
	}
	code := resp.StatusCode
	if code >= 200 && code < 300 {
		return nil
	}
	return fmt.Errorf("Unexpected DeleteFile (DELETE) code: %d", code)
}

func (c *Client) readTree() error {
	if c.treeItems != nil {
		return nil
	}
	bytes, err := c.requestBytes("GET", "/resources/files", url.Values{
		"limit": []string{"999999"},
	})
	if err != nil {
		return err
	}
	log.Debug("read tree: parsing json...")
	r := &Resources{}
	err = json.Unmarshal(bytes, &r)
	if err != nil {
		return err
	}

	c.treeParents = map[string]Resource{}
	c.treeParentPaths = []string{}
	c.treeItems = map[string]Resource{}
	c.treeItemPaths = []string{}

	log.Debug("read tree: filling tree and parents...")
	for _, item := range r.Items {
		c.appendTree(item)
	}

	log.Debug("read tree: sorting parent paths...")
	sort.Slice(c.treeParentPaths, func(i, j int) bool {
		return c.treeParentPaths[i] < c.treeParentPaths[j]
	})

	log.Debug("read tree: sorting item paths...")
	sort.Slice(c.treeItemPaths, func(i, j int) bool {
		return c.treeItemPaths[i] < c.treeItemPaths[j]
	})

	log.Debug("read tree: complete")
	return nil
}

func (c *Client) appendTree(item Resource) {
	absPath := item.GetNormalizedAbsPath()
	if strings.HasPrefix(absPath, c.opt.getBaseDir()) {
		path := c.opt.toRelPath(absPath)
		c.treeItems[path] = item
		c.treeItemPaths = append(c.treeItemPaths, path)

		// add missing dirs
		for _, dirPath := range util.PathParents(path) {
			if _, exists := c.treeItems[dirPath]; exists {
				continue
			}
			log.Debug("read tree: appending dir", dirPath)
			c.treeItems[dirPath] = c.createDirResource(c.opt.toAbsPath(dirPath))
			c.treeItemPaths = append(c.treeItemPaths, dirPath)
		}
	}

	// add missing parents
	for _, parentAbsPath := range util.PathParents(absPath) {
		if _, exists := c.treeParents[parentAbsPath]; exists {
			continue
		}
		if !strings.HasPrefix(c.opt.getBaseDir(), parentAbsPath) {
			continue
		}
		if parentAbsPath == absPath {
			c.treeParents[parentAbsPath] = item
			continue
		}
		log.Debug("read tree: appending parent", parentAbsPath)
		c.treeParents[parentAbsPath] = c.createDirResource(parentAbsPath)
		c.treeParentPaths = append(c.treeParentPaths, parentAbsPath)
	}
}

func (c *Client) createDirResource(absPath string) Resource {
	return Resource{
		Path: "disk:" + strings.TrimPrefix(absPath, "disk:"),
		Type: "dir",
	}
}

// http impl

func (c *Client) requestBytes(method, path string, query url.Values) ([]byte, error) {
	resp, err := c.request(method, path, query)
	if err != nil {
		return nil, err
	}
	log.Debug("requestBytes code", resp.StatusCode, resp.Status)
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
	log.Debugf("createRequest\n  path: %s\n  uri: %s\n  method: %s\n\n", path, uri, method)

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
