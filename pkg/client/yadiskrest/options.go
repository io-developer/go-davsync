package yadiskrest

import (
	"github.com/io-developer/go-davsync/pkg/util"
)

type Options struct {
	BaseDir       string
	ApiUri        string
	AuthToken     string
	AuthTokenType string
	AuthUser      string
	AuthPass      string
}

func (o *Options) getBaseDir() string {
	return util.PathNormalizeBaseDir(o.BaseDir)
}

func (o *Options) toRelPath(absPath string) string {
	return util.PathRel(absPath, o.BaseDir)
}

func (o *Options) toAbsPath(relPath string) string {
	return util.PathAbs(relPath, o.BaseDir)
}
