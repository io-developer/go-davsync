package webdav

import (
	"encoding/xml"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type Adapter struct {
	RetryLimit int
	RetryDelay time.Duration

	opt         Options
	httpClient  http.Client
	baseHeaders map[string]string
}

func NewAdapter(opt Options) *Adapter {
	return &Adapter{
		opt:        opt,
		httpClient: http.Client{},
		baseHeaders: map[string]string{
			"Content-Type":   "application/xml;charset=UTF-8",
			"Accept":         "application/xml,text/xml",
			"Accept-Charset": "utf-8",
			//"Accept-Encoding": "",
		},
		RetryLimit: 30,
		RetryDelay: 1 * time.Second,
	}
}

func (c *Adapter) buildURI(path string) string {
	return fmt.Sprintf(
		"%s/%s",
		strings.TrimRight(c.opt.DavUri, "/"),
		strings.TrimLeft(path, "/"),
	)
}
func (c *Adapter) createRequest(
	method string,
	path string,
	body io.Reader,
	headers map[string]string,
) (*http.Request, error) {
	uri := c.buildURI(path)

	log.Printf(
		"createRequest\n  path: %s\n  uri: %s\n  method: %s\n  headers: %#v\n\n",
		path,
		uri,
		method,
		headers,
	)

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

func (c *Adapter) auth(req *http.Request) *http.Request {
	if c.opt.AuthToken != "" {
		authPrefix := c.opt.AuthTokenType
		if authPrefix == "" {
			authPrefix = "OAuth"
		}
		val := fmt.Sprintf("%s %s", authPrefix, c.opt.AuthToken)
		req.Header.Add("Authorization", val)
	} else {
		req.SetBasicAuth(c.opt.AuthUser, c.opt.AuthPass)
	}
	return req
}

func (c *Adapter) request(req *http.Request) (resp *http.Response, err error) {
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

func (c *Adapter) Propfind(path string, depth string) (result PropfindSome, code int, err error) {
	log.Println("Client.reqPropfind(): ", path, depth)

	reqBody := strings.NewReader(
		"<d:propfind xmlns:d='DAV:'>" +
			"<d:allprop/>" +
			"</d:propfind>",
	)
	req, err := c.createRequest("PROPFIND", path, reqBody, map[string]string{
		"Depth": depth,
	})
	if err != nil {
		return
	}
	resp, err := c.request(req)
	if err != nil {
		return
	}
	code = resp.StatusCode
	if code < 200 || code >= 300 {
		err = fmt.Errorf("Unexpected PROPFIND code %d '%s'", code, resp.Status)
		return
	}
	bytes, err := ioutil.ReadAll(resp.Body)
	log.Println("  response: ", string(bytes))

	err = xml.Unmarshal(bytes, &result)
	return
}

func (c *Adapter) Mkcol(path string) (code int, err error) {
	req, err := c.createRequest("MKCOL", path, nil, map[string]string{})
	if err != nil {
		return
	}
	resp, err := c.request(req)
	if err != nil {
		return
	}
	code = resp.StatusCode
	if code != 201 {
		err = fmt.Errorf("Expected MKCOL code 201, got %d '%s'", code, resp.Status)
	}
	return
}

func (c *Adapter) GetFile(path string) (r io.ReadCloser, code int, err error) {
	req, err := c.createRequest("GET", path, nil, map[string]string{})
	if err != nil {
		return
	}
	resp, err := c.request(req)
	if err != nil {
		return
	}
	r = resp.Body
	code = resp.StatusCode
	return
}

func (c *Adapter) PutFile(path string, body io.Reader) (code int, err error) {
	req, err := c.createRequest("PUT", path, body, map[string]string{})
	if err != nil {
		return
	}
	resp, err := c.request(req)
	if err != nil {
		return
	}
	code = resp.StatusCode
	return
}

func (c *Adapter) MoveFile(srcPath, dstPath string) (code int, err error) {
	req, err := c.createRequest("MOVE", srcPath, nil, map[string]string{
		"Destination": c.buildURI(url.PathEscape(dstPath)),
	})
	if err != nil {
		return
	}
	resp, err := c.request(req)
	if err != nil {
		return
	}
	code = resp.StatusCode
	return
}
