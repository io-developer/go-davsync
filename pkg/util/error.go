package util

import (
	"fmt"
	"io"
	"net/url"
)

func ErrorIsEOF(err error) bool {
	if err == nil {
		return false
	}
	if err == io.EOF {
		fmt.Println("isErrEOF: io.EOF")
		return true
	}
	if err.Error() == "EOF" {
		fmt.Println("isErrEOF: 'EOF'")
		return true
	}
	uerr, isURL := err.(*url.Error)
	if isURL && uerr.Err == io.EOF {
		fmt.Println("isErrEOF: isURL io.EOF")
		return true
	}
	if isURL && uerr.Err.Error() == "EOF" {
		fmt.Println("isErrEOF: isURL 'EOF'")
		return true
	}
	return false
}
