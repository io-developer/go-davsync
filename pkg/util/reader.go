package util

import (
	"crypto"
	"fmt"
	"hash"
	"io"
)

type Reader struct {
	io.ReadCloser

	OnProgress func(r *Reader)
	OnComplete func(r *Reader)

	reader     io.ReadCloser
	bytesTotal int64
	bytesRead  int64
	isComplete bool
	md5        hash.Hash
	sha256     hash.Hash
}

func NewRead(r io.ReadCloser, len int64) *Reader {
	return &Reader{
		reader:     r,
		bytesTotal: len,
		bytesRead:  0,
		md5:        crypto.MD5.New(),
		sha256:     crypto.SHA256.New(),
	}
}

func (r *Reader) IsComplete() bool {
	return r.isComplete
}

func (r *Reader) GetBytesRead() int64 {
	return r.bytesRead
}

func (r *Reader) GetBytesTotal() int64 {
	return r.bytesTotal
}

func (r *Reader) GetProgress() float64 {
	if r.bytesTotal <= 0 || r.bytesRead <= 0 {
		return 0
	}
	return float64(r.bytesRead) / float64(r.bytesTotal)
}

func (r *Reader) GetHashMd5() string {
	return fmt.Sprintf("%x", r.md5.Sum(nil))
}

func (r *Reader) GetHashSha256() string {
	return fmt.Sprintf("%x", r.sha256.Sum(nil))
}

func (r *Reader) Read(p []byte) (n int, err error) {
	n, err = r.reader.Read(p)
	r.bytesRead += int64(n)
	r.updateHash(p, n)

	isCompletePrev := r.isComplete
	r.isComplete = r.bytesRead >= r.bytesTotal

	if r.OnProgress != nil {
		r.OnProgress(r)
	}
	if r.OnComplete != nil && r.isComplete && !isCompletePrev {
		r.OnComplete(r)
	}
	return
}

func (r *Reader) Close() error {
	return r.reader.Close()
}

func (r *Reader) updateHash(p []byte, n int) error {
	if n < 1 {
		return nil
	}

	data := make([]byte, n)
	copy(data, p)

	nMd5, err := r.md5.Write(data)
	if err != nil {
		return err
	}
	if nMd5 != n {
		return fmt.Errorf("ReadProgress: n md5 (%d) != n (%d)", nMd5, n)
	}

	nSha256, err := r.sha256.Write(data)
	if err != nil {
		return err
	}
	if nMd5 != n {
		return fmt.Errorf("ReadProgress: n sha256 (%d) != n (%d)", nSha256, n)
	}

	return nil
}
