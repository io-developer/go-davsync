package yadiskrest

import "github.com/io-developer/go-davsync/pkg/client"

type Options struct {
	BaseDir       string
	ApiUri        string
	AuthToken     string
	AuthTokenType string
	AuthUser      string
	AuthPass      string
}

func (o *Options) getBaseDir() string {
	return client.PathNormalizeBaseDir(o.BaseDir)
}

func (o *Options) toRelPath(absPath string) string {
	return client.PathRel(absPath, o.BaseDir)
}

func (o *Options) toAbsPath(relPath string) string {
	return client.PathAbs(relPath, o.BaseDir)
}