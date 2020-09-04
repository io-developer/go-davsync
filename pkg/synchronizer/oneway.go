package synchronizer

import (
	"crypto"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/io-developer/go-davsync/pkg/client"
	"github.com/io-developer/go-davsync/pkg/util"
)

type OneWayOpt struct {
	IgnoreExisting         bool
	IndirectUpload         bool
	UploadPathFormat       string
	AllowDelete            bool
	SingleThreadedFileSize int64
	ThreadCount            uint
	AttemptMax             uint
	AttemptDelay           time.Duration
	UploadCheckTimeout     time.Duration
	UploadCheckDelay       time.Duration
}

type OneWay struct {
	opt    OneWayOpt
	input  client.Client
	output client.Client

	inputTree  *client.TreeBuffer
	outputTree *client.TreeBuffer

	bothPaths []string
	addPaths  []string
	delPaths  []string

	signleThreadUpload sync.Mutex
}

func NewOneWay(input, output client.Client, opt OneWayOpt) *OneWay {
	if opt.UploadPathFormat == "" {
		opt.UploadPathFormat = "/ucam-%x.bin"
	}
	if opt.ThreadCount < 1 {
		opt.ThreadCount = 1
	}
	if opt.AttemptMax < 1 {
		opt.AttemptMax = 1
	}
	if opt.UploadCheckTimeout < time.Second {
		opt.UploadCheckTimeout = time.Second
	}
	if opt.UploadCheckDelay < time.Second {
		opt.UploadCheckDelay = time.Second
	}
	return &OneWay{
		opt:                opt,
		input:              input,
		output:             output,
		inputTree:          client.NewTreeBuffer(input),
		outputTree:         client.NewTreeBuffer(output),
		signleThreadUpload: sync.Mutex{},
	}
}

func (s *OneWay) Sync(errors chan<- error) {
	s.readTrees(errors)
	s.calcDiff()

	s.makeDirs(errors)
	s.handlePaths(s.addPaths, s.uploadFile, "UPL", errors)

	if s.opt.AllowDelete {
		s.handlePaths(s.delPaths, s.deleteOutputFile, "DEL", errors)
	}
}

func (s *OneWay) log(msg string) {
	log.Printf("Sync: %s\n", msg)
}

func (s *OneWay) readTrees(errors chan<- error) {
	group := sync.WaitGroup{}
	group.Add(2)
	go func() {
		if err := s.inputTree.Read(); err != nil {
			errors <- err
		}
		group.Done()
	}()
	go func() {
		if err := s.outputTree.Read(); err != nil {
			errors <- err
		}
		group.Done()
	}()
	group.Wait()
}

func (s *OneWay) calcDiff() {
	s.log("Calculating input/output path diff...")

	s.bothPaths, s.addPaths, s.delPaths = util.Diff(
		s.inputTree.GetChildrenPaths(),
		s.outputTree.GetChildrenPaths(),
	)

	s.log("Path diff:")
	for _, path := range s.bothPaths {
		s.log(fmt.Sprintf("BOTH %s", path))
	}
	for _, path := range s.addPaths {
		s.log(fmt.Sprintf("ADD %s", path))
	}
	for _, path := range s.delPaths {
		s.log(fmt.Sprintf("DEL %s", path))
	}
}

func (s *OneWay) makeDirs(errors chan<- error) {
	s.log("Making dirs...")

	bothPathDirs := util.PathSortedDirs(s.bothPaths)
	addPathDirs := util.PathSortedDirs(s.addPaths)
	_, addDirs, _ := util.Diff(addPathDirs, bothPathDirs)
	addDirs = util.PathSortedDirs(addDirs)

	for _, path := range addDirs {
		s.log(fmt.Sprintf("  make dir %s", path))

		err := s.outputTree.MakeDir(path, true)
		if err != nil {
			errors <- err
		}
	}
}

func (s *OneWay) handlePaths(
	paths []string,
	handler func(path string, logFn func(msg string)) error,
	logPrefix string,
	errors chan<- error,
) {
	total := 0
	handled := 0
	logMain := func(msg string) {
		progress := 0.0
		if total > 0 {
			progress = 100.0 * float64(handled) / float64(total)
		}
		s.log(fmt.Sprintf("%s %.2f%% (%d/%d): %s", logPrefix, progress, handled, total, msg))
	}

	logMain("Handling...")
	if len(paths) == 0 {
		logMain("Nothing to do")
		return
	}

	sortedPaths := util.PathSorted(paths)
	total = len(sortedPaths)

	pathsCh := make(chan string)
	group := sync.WaitGroup{}

	thread := func(id uint) {
		curPath := "-"
		logThread := func(msg string) {
			logMain(fmt.Sprintf("[t%d] '%s': %s", id, curPath, msg))
		}
		logThread("Thread started")

		for {
			select {
			case path, ok := <-pathsCh:
				if !ok {
					logThread("Thread exited")
					group.Done()
					return
				}
				curPath = path
				var handleErr error = nil
				for i := uint(1); i <= s.opt.AttemptMax; i++ {
					if i > 1 || i == s.opt.AttemptMax {
						logThread(fmt.Sprintf("Attempt %d / %d", i, s.opt.AttemptMax))
					}
					handleErr = handler(path, logThread)
					if handleErr == nil {
						break
					}
					logThread(fmt.Sprintf("Attempt %d / %d ERR: '%v'", i, s.opt.AttemptMax, handleErr))
					time.Sleep(s.opt.AttemptDelay)
				}
				if handleErr != nil {
					logThread(fmt.Sprintf("ERROR '%v'", handleErr))
					errors <- handleErr
				} else {
					logThread("Complete")
				}
				handled++
				curPath = "-"
			}
		}
	}

	for i := uint(0); i < s.opt.ThreadCount; i++ {
		group.Add(1)
		go thread(i)
	}
	for _, path := range sortedPaths {
		pathsCh <- path
	}
	close(pathsCh)

	group.Wait()

	logMain("Complete")
}

func (s *OneWay) uploadFile(path string, logFn func(string)) error {
	res, exists := s.inputTree.GetChild(path)
	if !exists {
		logFn("Not exists. Skiping..")
		return nil
	}
	if res.IsDir {
		logFn("Direcory. Skiping..")
		return nil
	}

	isSingleThreadLocked := true
	s.signleThreadUpload.Lock()

	if !s.isSingleThreadUploadNeeded(res) {
		isSingleThreadLocked = false
		s.signleThreadUpload.Unlock()
	} else {
		logFn("Single-thread upload begin..")
	}

	unlockIfNeeded := func() {
		if isSingleThreadLocked {
			isSingleThreadLocked = false

			logFn("Single-thread upload end")
			s.signleThreadUpload.Unlock()
		}
	}

	uploadPath := s.getUploadPath(path, res, s.opt.IndirectUpload)
	logFn(fmt.Sprintf("Uploading to '%s'", uploadPath))

	inputReader, err := s.input.ReadFile(path)
	if err != nil {
		unlockIfNeeded()
		return err
	}

	logRead := func(r *util.Reader) {
		logFn(fmt.Sprintf(
			"%.2f%% (%s / %s)",
			100*r.GetProgress(),
			util.FormatBytes(r.GetBytesRead()),
			util.FormatBytes(r.GetBytesTotal()),
		))
	}
	readerLogInterval := 2 * time.Second
	readerLogLastTime := time.Now()

	reader := util.NewRead(inputReader, res.Size)
	reader.OnProgress = func(r *util.Reader) {
		if time.Now().Sub(readerLogLastTime) >= readerLogInterval {
			readerLogLastTime = time.Now()
			logRead(r)
		}
	}
	reader.OnComplete = func(r *util.Reader) {
		logRead(r)
		unlockIfNeeded()
	}

	err = s.output.WriteFile(uploadPath, reader, res.Size)
	time.Sleep(time.Second)

	unlockIfNeeded()
	reader.Close()

	logFn(fmt.Sprintf("Reader IsComplete %t", reader.IsComplete()))
	logFn(fmt.Sprintf("Reader err is EOF %t", util.ErrorIsEOF(err)))

	if err != nil && !util.ErrorIsEOF(err) {
		return err
	}

	logFn(fmt.Sprintf("Read bytes: %d", reader.GetBytesRead()))
	logFn(fmt.Sprintf("Read md5: %s", reader.GetHashMd5()))
	logFn(fmt.Sprintf("Read sha256: %s", reader.GetHashSha256()))

	logFn(fmt.Sprintf("Checking %s", uploadPath))
	err = s.checkUploaded(uploadPath, res, reader, logFn)
	if err != nil {
		return err
	}

	if path != uploadPath {
		logFn(fmt.Sprintf("Moving %s", uploadPath))
		err = s.output.MoveFile(uploadPath, path)
		if err != nil {
			return err
		}
	}

	return nil
}

func (s *OneWay) isSingleThreadUploadNeeded(res client.Resource) bool {
	if s.opt.SingleThreadedFileSize <= 0 {
		return false
	}
	return res.Size > s.opt.SingleThreadedFileSize
}

func (s *OneWay) getUploadPath(path string, res client.Resource, indirect bool) string {
	if !indirect {
		return path
	}
	sign := fmt.Sprintf("%s:%d", path, res.Size)
	h := crypto.SHA256.New()
	h.Write([]byte(sign))
	return fmt.Sprintf(s.opt.UploadPathFormat, h.Sum(nil))
}

func (s *OneWay) checkUploaded(
	path string,
	res client.Resource,
	r *util.Reader,
	logFn func(string),
) (err error) {
	if !r.IsComplete() {
		return fmt.Errorf(
			"Upload not complete: %d of %d (%s / %s)",
			r.GetBytesRead(),
			r.GetBytesTotal(),
			util.FormatBytes(r.GetBytesRead()),
			util.FormatBytes(r.GetBytesTotal()),
		)
	}
	timeout := s.opt.UploadCheckTimeout
	timeStart := time.Now()
	for time.Now().Sub(timeStart) < timeout {
		logFn(fmt.Sprintf(
			"Checking (%s / %s) '%s'",
			time.Now().Sub(timeStart).String(),
			timeout.String(),
			path,
		))
		written, isExist, resErr := s.output.GetResource(path)
		err = resErr
		if err == nil && isExist {
			err = s.checkUploadedRes(path, res, written, r, logFn)
			if err == nil {
				return
			}
		}
		time.Sleep(s.opt.UploadCheckDelay)
	}
	if err != nil {
		return err
	}
	return fmt.Errorf("File uploaded but not found atfer timeout %s", timeout.String())
}

func (s *OneWay) checkUploadedRes(
	path string,
	input, uploaded client.Resource,
	r *util.Reader,
	logFn func(string),
) (err error) {
	if uploaded.HashSha256 != "" {
		if uploaded.HashSha256 == r.GetHashSha256() {
			logFn("Check OK: SHA256 strict matched")
			return nil
		}
		logFn("Check FAIL: SHA256 not matched")
		return fmt.Errorf(
			"Written SHA256 not matched (%s -> %s), %s",
			r.GetHashSha256(),
			uploaded.HashSha256,
			path,
		)
	}
	if uploaded.HashMd5 != "" {
		if uploaded.HashMd5 == r.GetHashMd5() {
			logFn("Check OK: MD5 strict matched")
			return nil
		}
		logFn("Check FAIL: MD5 not matched")
		return fmt.Errorf(
			"Written MD5 not matched (%s -> %s), %s",
			r.GetHashMd5(),
			uploaded.HashMd5,
			path,
		)
	}
	if uploaded.MatchAnyHash(r.GetHashSha256()) {
		logFn("Check OK: SHA256 matched")
		return nil
	}
	if uploaded.MatchAnyHash(r.GetHashMd5()) {
		logFn("Check OK: MD5 matched")
		return nil
	}
	if uploaded.Size == input.Size && input.Size == r.GetBytesRead() {
		logFn("Check OK: size matched")
		return nil
	}
	logFn("Check FAIL: size not matched")
	return fmt.Errorf(
		"Uploaded size not matched (%d -> %d), %s",
		input.Size,
		uploaded.Size,
		path,
	)
}

func (s *OneWay) deleteOutputFile(path string, logFn func(string)) error {
	res, exists := s.outputTree.GetChild(path)
	if !exists {
		logFn("Not exists. Skiping..")
		return nil
	}
	if res.IsDir {
		logFn("Direcory. Skiping..")
		return nil
	}
	return s.output.DeleteFile(path)
}
