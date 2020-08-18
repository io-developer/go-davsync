package yadiskrest

import (
	"encoding/json"
	"time"
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
	Sha256     string    `json:"sha256"`
}

func (r Resource) IsDir() bool {
	return r.Type == "dir"
}

type UploadInfo struct {
	Href      string `json:"href"`
	Method    string `json:"method"`
	Templated bool   `json:"templated"`
}

func ResourcesParse(jsonBytes []byte) (Resources, error) {
	r := Resources{}
	err := json.Unmarshal(jsonBytes, &r)
	return r, err
}
