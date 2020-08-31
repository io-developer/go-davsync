package webdav

import (
	"github.com/io-developer/go-davsync/pkg/util"
)

type Options struct {
	BaseDir       string
	DavUri        string
	AuthToken     string
	AuthTokenType string
	AuthUser      string
	AuthPass      string
}

func (o *Options) toRelPath(absPath string) string {
	return util.PathRel(absPath, o.BaseDir)
}

func (o *Options) toAbsPath(relPath string) string {
	return util.PathAbs(relPath, o.BaseDir)
}
