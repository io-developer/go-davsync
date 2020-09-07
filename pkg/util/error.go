package util

import (
	"io"
	"net/url"

	"github.com/io-developer/go-davsync/pkg/log"
)

func ErrorIsEOF(err error) bool {
	if err == nil {
		return false
	}
	if err == io.EOF {
		log.Debug("isErrEOF: io.EOF")
		return true
	}
	if err.Error() == "EOF" {
		log.Debug("isErrEOF: 'EOF'")
		return true
	}
	uerr, isURL := err.(*url.Error)
	if isURL && uerr.Err == io.EOF {
		log.Debug("isErrEOF: isURL io.EOF")
		return true
	}
	if isURL && uerr.Err.Error() == "EOF" {
		log.Debug("isErrEOF: isURL 'EOF'")
		return true
	}
	return false
}
