package webdav

import (
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/io-developer/davsync/model"
)

type Client struct {
	BaseURI       string
	BasePath      string
	AuthToken     string
	AuthTokenType string
	AuthUser      string
	AuthPass      string
	RetryLimit    int
	RetryDelay    time.Duration
	httpClient    http.Client
	baseHeaders   map[string]string
}

func NewClient() *Client {
	return &Client{
		httpClient: http.Client{},
		baseHeaders: map[string]string{
			"Content-Type":   "application/xml;charset=UTF-8",
			"Accept":         "application/xml,text/xml",
			"Accept-Charset": "utf-8",
			//"Accept-Encoding": "",
		},
		RetryLimit: 3,
		RetryDelay: 1 * time.Second,
	}
}

func (c *Client) ReadTree() (paths []string, nodes map[string]model.Node, err error) {
	paths = []string{}
	nodes = map[string]model.Node{}
	err = c.readTree("/", &paths, nodes)
	return
}

func (c *Client) readTree(
	path string,
	outPaths *[]string,
	outNodes map[string]model.Node,
) (err error) {
	some, err := c.PropfindSome(path, 1)
	if err != nil {
		return
	}
	for _, item := range some.Propfinds {
		absPath := item.GetHrefUnicode()
		relPath := strings.TrimPrefix(absPath, c.BasePath)
		if _, exists := outNodes[relPath]; exists {
			continue
		}
		*outPaths = append(*outPaths, relPath)
		outNodes[relPath] = model.Node{
			AbsPath:  absPath,
			Path:     relPath,
			Name:     item.DisplayName,
			IsDir:    item.IsCollection(),
			Size:     item.ContentLength,
			UserData: item,
		}
		if item.IsCollection() && relPath != path {
			c.readTree(relPath, outPaths, outNodes)
		}
	}
	return
}

func (c *Client) createRequest(method, path string, body io.Reader, headers map[string]string) (*http.Request, error) {
	uri := c.buildURI(path)
	log.Printf("createRequest\n  path: %s\n  uri: %s\n  method: %s\n\n", path, uri, method)

	req, err := http.NewRequest(method, uri, body)
	if err != nil {
		return req, err
	}
	for key, val := range c.baseHeaders {
		req.Header.Add(key, val)
	}
	for key, val := range headers {
		req.Header.Add(key, val)
	}
	return c.auth(req), err
}

func (c *Client) buildURI(path string) string {
	uriPath := filepath.Join(c.BasePath, path)
	return fmt.Sprintf("%s/%s", strings.TrimRight(c.BaseURI, "/"), strings.TrimLeft(uriPath, "/"))
}

func (c *Client) auth(req *http.Request) *http.Request {
	if c.AuthToken != "" {
		authPrefix := c.AuthTokenType
		if authPrefix == "" {
			authPrefix = "OAuth"
		}
		req.Header.Add("Authorization", fmt.Sprintf("%s %s", authPrefix, c.AuthToken))
	} else {
		req.SetBasicAuth(c.AuthUser, c.AuthPass)
	}
	return req
}

func (c *Client) request(req *http.Request) (resp *http.Response, err error) {
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

func (c *Client) requestBytes(req *http.Request) ([]byte, error) {
	resp, err := c.request(req)
	if err != nil {
		return nil, err
	}
	return ioutil.ReadAll(resp.Body)
}

func (c *Client) Propfind(path string) (result Propfind, err error) {
	items, err := c.PropfindSome(path, 0)
	if err == nil {
		if len(items.Propfinds) == 1 {
			result = items.Propfinds[0]
		} else {
			err = errors.New(fmt.Sprintf("Expected one propfind, got %d", len(items.Propfinds)))
		}
	}
	return
}

func (c *Client) PropfindSome(path string, depth int) (result PropfindSome, err error) {
	if bytes, err := c.reqPropfind(path, depth); err == nil {
		err = xml.Unmarshal(bytes, &result)
	}
	return
}

func (c *Client) reqPropfind(path string, depth int) (bytes []byte, err error) {
	reqBody := strings.NewReader(
		"<d:propfind xmlns:d='DAV:'>" +
			"<d:allprop/>" +
			"</d:propfind>",
	)
	req, err := c.createRequest("PROPFIND", path, reqBody, map[string]string{
		"Depth": strconv.Itoa(depth),
	})
	log.Println("Client.reqPropfind(): ", path, depth)
	if err != nil {
		return
	}
	bytes, err = c.requestBytes(req)
	log.Println("  response: ", string(bytes))

	return
}
