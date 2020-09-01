package client

import (
	"os"
	"time"
)

type Resource struct {
	Name       string
	Path       string
	AbsPath    string
	IsDir      bool
	Size       int64
	ModTime    time.Time
	HashETag   string
	HashMd5    string
	HashSha256 string
	UserData   interface{}
}

func (r Resource) IsLocal() bool {
	_, ok := r.UserData.(os.FileInfo)
	return ok
}

func (r Resource) MatchAnyHash(h string) bool {
	if r.HashSha256 != "" && r.HashSha256 == h {
		return true
	}
	if r.HashMd5 != "" && r.HashMd5 == h {
		return true
	}
	if r.HashETag != "" && r.HashETag == h {
		return true
	}
	return false
}
