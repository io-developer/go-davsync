package webdav

import (
	"encoding/xml"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"strings"
)

type Client struct {
	BaseURI       string
	AuthToken     string
	AuthTokenType string
	AuthUser      string
	AuthPass      string
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
	}
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

func (c *Client) request(req *http.Request) (*http.Response, error) {
	return c.httpClient.Do(req)
}

func (c *Client) requestBytes(req *http.Request) ([]byte, error) {
	resp, err := c.request(req)
	if err != nil {
		return nil, err
	}
	return ioutil.ReadAll(resp.Body)
}

func (c *Client) Propfind(path string) (*PropfindMultistatus, error) {
	result := &PropfindMultistatus{}

	reqBody := strings.NewReader(
		"<d:propfind xmlns:d='DAV:'>" +
			"<d:allprop/>" +
			"</d:propfind>",
	)
	req, err := c.createRequest("PROPFIND", path, reqBody, map[string]string{
		"Depth": "1",
	})
	if err != nil {
		return result, err
	}

	respBytes, err := c.requestBytes(req)
	if err != nil {
		return result, err
	}

	log.Println("Client.Propfind() respBytes: ", string(respBytes))

	if err = xml.Unmarshal(respBytes, result); err != nil {
		return result, err
	}

	return result, err
}
