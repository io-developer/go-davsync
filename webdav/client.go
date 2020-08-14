package webdav

import (
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
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
		itemPath := item.Href
		if _, exists := outNodes[itemPath]; exists {
			continue
		}
		*outPaths = append(*outPaths, itemPath)
		outNodes[itemPath] = model.Node{
			AbsPath: item.Href,
			Path:    itemPath,
			Name:    item.DisplayName,
			IsDir:   item.IsCollection(),
			Size:    item.ContentLength,
		}
		if item.IsCollection() && itemPath != path {
			c.readTree(itemPath, outPaths, outNodes)
		}
	}
	return
}

func (c *Client) createRequest(method, path string, body io.Reader, headers map[string]string) (*http.Request, error) {
	req, err := http.NewRequest(method, c.buildURI(path), body)
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
	return fmt.Sprintf("%s/%s", strings.TrimRight(c.BaseURI, "/"), strings.Trim(path, "/"))
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
	if err != nil {
		return
	}
	if len(items.Propfinds) != 1 {
		errors.New(fmt.Sprintf("Expected one propfind, got %d", len(items.Propfinds)))
	}
	return items.Propfinds[0], nil
}

func (c *Client) PropfindSome(path string, depth int) (result PropfindSome, err error) {
	if bytes, err := c.reqPropfind(path, depth); err == nil {
		err = xml.Unmarshal(bytes, &result)
	}
	return
}

func (c *Client) reqPropfind(path string, depth int) ([]byte, error) {
	reqBody := strings.NewReader(
		"<d:propfind xmlns:d='DAV:'>" +
			"<d:allprop/>" +
			"</d:propfind>",
	)
	req, err := c.createRequest("PROPFIND", path, reqBody, map[string]string{
		"Depth": strconv.Itoa(depth),
	})
	if err != nil {
		return nil, err
	}
	bytes, err := c.requestBytes(req)
	log.Println("Client.reqPropfind(): ", depth, string(bytes))

	return bytes, err
}
