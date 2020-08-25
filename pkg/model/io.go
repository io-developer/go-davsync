package model

import (
	"fmt"
	"io"
	"time"
)

type ReadProgress struct {
	io.ReadCloser

	LogInterval time.Duration

	reader      io.ReadCloser
	bytesTotal  int64
	bytesRead   int64
	isComplete  bool
	logFn       func(string)
	logLastTime time.Time
}

func NewReadProgress(r io.ReadCloser, len int64) *ReadProgress {
	return &ReadProgress{
		reader:      r,
		bytesTotal:  len,
		bytesRead:   0,
		LogInterval: 2 * time.Second,
		logFn:       nil,
		logLastTime: time.Now(),
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

	isComplete := r.bytesRead == r.bytesTotal
	if !isComplete {
		r.Log(false)
	} else if !r.isComplete {
		r.Log(true)
	}
	r.isComplete = isComplete

	return
}

func (r *ReadProgress) SetLogFn(f func(string)) {
	r.logFn = f
}

func (r *ReadProgress) Log(force bool) {
	if r.logFn == nil {
		return
	}
	isTime := time.Now().Sub(r.logLastTime) >= r.LogInterval
	if force || isTime {
		r.logLastTime = time.Now()
		r.logFn(fmt.Sprintf(
			"%.2f%% (%d of %d)",
			100*r.GetProgress(),
			r.bytesRead,
			r.bytesTotal,
		))
	}
}

func (r *ReadProgress) Close() error {
	return r.reader.Close()
}
