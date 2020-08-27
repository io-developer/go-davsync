package model

import (
	"crypto"
	"fmt"
	"hash"
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
	md5         hash.Hash
	sha256      hash.Hash
}

func NewReadProgress(r io.ReadCloser, len int64) *ReadProgress {
	return &ReadProgress{
		reader:      r,
		bytesTotal:  len,
		bytesRead:   0,
		LogInterval: 2 * time.Second,
		logFn:       nil,
		logLastTime: time.Now(),
		md5:         crypto.MD5.New(),
		sha256:      crypto.SHA256.New(),
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

func (r *ReadProgress) GetHashMd5() string {
	return fmt.Sprintf("%x", r.md5.Sum(nil))
}

func (r *ReadProgress) GetHashSha256() string {
	return fmt.Sprintf("%x", r.sha256.Sum(nil))
}

func (r *ReadProgress) Read(p []byte) (n int, err error) {
	n, err = r.reader.Read(p)
	r.bytesRead += int64(n)
	r.updateHash(p, n)

	isComplete := r.bytesRead == r.bytesTotal
	if !isComplete {
		r.Log(false)
	} else if !r.isComplete {
		r.Log(true)
	}
	r.isComplete = isComplete

	return
}

func (r *ReadProgress) updateHash(p []byte, n int) error {
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
			"%.2f%% (%s / %s)",
			100*r.GetProgress(),
			formatBytes(r.bytesRead),
			formatBytes(r.bytesTotal),
		))
	}
}

func (r *ReadProgress) Close() error {
	return r.reader.Close()
}

func formatBytes(size int64) string {
	if size < 1024 {
		return fmt.Sprintf("%d B", size)
	}
	rest := size
	mul := uint64(1)
	exp := uint64(0)
	for (rest >> 10) > 0 {
		rest = rest >> 10
		mul = mul << 10
		exp++
	}
	val := float64(size) / float64(mul)
	suffixes := []string{"B", "KiB", "MiB", "GiB", "TiB", "PiB", "EiB"}
	return fmt.Sprintf("%.1f %s", val, suffixes[exp])
}
