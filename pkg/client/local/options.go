package local

import (
	"os"

	"github.com/io-developer/go-davsync/pkg/util"
)

type Options struct {
	BaseDir  string
	DirMode  os.FileMode
	FileMode os.FileMode
}

func (o *Options) toRelPath(absPath string) string {
	return util.PathRel(absPath, o.BaseDir)
}

func (o *Options) toAbsPath(relPath string) string {
	return util.PathAbs(relPath, o.BaseDir)
}
