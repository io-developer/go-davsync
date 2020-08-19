package yadiskrest

import (
	"time"

	"github.com/io-developer/go-davsync/pkg/client"
)

type Resources struct {
	Items []Resource `json:"items"`
}

type Resource struct {
	ResourceID string    `json:"resource_id"`
	Path       string    `json:"path"`
	Type       string    `json:"type"`
	MediaType  string    `json:"media_type"`
	MimeType   string    `json:"mime_type"`
	Created    time.Time `json:"created"`
	Modified   time.Time `json:"modified"`
	Name       string    `json:"name"`
	File       string    `json:"file"`
	Size       int64     `json:"size"`
	Md5        string    `json:"md5"`
	Sha256     string    `json:"sha256,omitempty"`
}

func (r Resource) IsFile() bool {
	return r.Type == "file"
}

func (r Resource) IsDir() bool {
	return r.Type == "dir"
}

func (r Resource) GetNormalizedAbsPath() string {
	return client.NormalizePath(r.Path, r.IsDir())
}

type UploadInfo struct {
	Href      string `json:"href"`
	Method    string `json:"method"`
	Templated bool   `json:"templated"`
}
