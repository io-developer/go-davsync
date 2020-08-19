package model

import (
	"io"
	"log"
)

type ReadProgress struct {
	io.ReadCloser

	reader io.ReadCloser

	bytesTotal int64
	bytesRead  int64
}

func NewReadProgress(r io.ReadCloser, len int64) *ReadProgress {
	return &ReadProgress{
		reader:     r,
		bytesTotal: len,
		bytesRead:  0,
	}
}

func (r *ReadProgress) GetBytesRead() int64 {
	return r.bytesRead
}

func (r *ReadProgress) GetBytesTotal() int64 {
	return r.bytesTotal
}

func (r *ReadProgress) GetProgress() float64 {
	if r.bytesTotal <= 0 || r.bytesRead <= 0 {
		return 0
	}
	return float64(r.bytesRead) / float64(r.bytesTotal)
}

func (r *ReadProgress) Read(p []byte) (n int, err error) {
	n, err = r.reader.Read(p)
	r.bytesRead += int64(n)

	log.Printf(
		"Read progress %.2f%%. n: %d, read: %d, total: %d",
		100*r.GetProgress(),
		n,
		r.bytesRead,
		r.bytesTotal,
	)

	return
}

func (r *ReadProgress) Close() error {
	return r.reader.Close()
}
