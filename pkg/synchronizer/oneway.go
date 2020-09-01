package synchronizer

import (
	"crypto"
	"fmt"
	"io"
	"log"
	"net/url"
	"regexp"
	"sort"
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
	WriteThreads           uint
	WriteRetry             uint
	WriteRetryDelay        time.Duration
	WriteCheckTimeout      time.Duration
	WriteCheckDelay        time.Duration
}

type OneWay struct {
	src client.Client
	dst client.Client
	opt OneWayOpt

	// sync-time data
	srcPaths     []string
	srcResources map[string]client.Resource

	dstPaths []string
	dstNodes map[string]client.Resource

	bothPaths []string
	addPaths  []string
	delPaths  []string

	signleWriteMutex sync.Mutex
}

func NewOneWay(src, dst client.Client, opt OneWayOpt) *OneWay {
	if opt.UploadPathFormat == "" {
		opt.UploadPathFormat = "/ucam-%x.bin"
	}
	if opt.WriteThreads < 1 {
		opt.WriteThreads = 1
	}
	if opt.WriteRetry < 1 {
		opt.WriteRetry = 1
	}
	if opt.WriteCheckTimeout < time.Second {
		opt.WriteCheckTimeout = time.Second
	}
	if opt.WriteCheckDelay < time.Second {
		opt.WriteCheckDelay = time.Second
	}
	return &OneWay{
		src:              src,
		dst:              dst,
		opt:              opt,
		signleWriteMutex: sync.Mutex{},
	}
}

func (s *OneWay) Sync(errors chan<- error) {
	s.readTrees(errors)
	s.logTrees()

	s.diff()
	s.makeDirs(errors)
	s.writeFiles(errors)
}

func (s *OneWay) log(msg string) {
	log.Printf("Sync: %s\n", msg)
}

func (s *OneWay) readTrees(errors chan<- error) {
	group := sync.WaitGroup{}
	group.Add(2)
	go func() {
		var err error
		s.srcPaths, s.srcResources, err = s.src.ReadTree()
		if err != nil {
			errors <- err
		}
		group.Done()
	}()
	go func() {
		var err error
		s.dstPaths, s.dstNodes, err = s.dst.ReadTree()
		if err != nil {
			errors <- err
		}
		group.Done()
	}()
	group.Wait()
}

func (s *OneWay) logTrees() {
	s.log("")
	s.log("Source paths:")
	for _, path := range s.srcPaths {
		s.log(path)
		res := s.srcResources[path]
		s.log(fmt.Sprintf("ModTime: %s", res.ModTime.Format("2006-01-02 15:04:05 -0700")))
	}
	s.log("")

	s.log("")
	s.log("Destination paths:")
	for _, path := range s.dstPaths {
		s.log(path)
		res := s.dstNodes[path]
		s.log(fmt.Sprintf("ModTime: %s", res.ModTime.Format("2006-01-02 15:04:05 -0700")))
	}
	s.log("")
}

func (s *OneWay) diff() {
	s.log("Comparing trees...")

	from := []string{}
	for path := range s.srcResources {
		from = append(from, path)
	}
	to := []string{}
	for path := range s.dstNodes {
		to = append(to, path)
	}
	s.bothPaths, s.addPaths, s.delPaths = diff(from, to)

	s.log("Tree diff:")
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

	bothPathDirs := getSortedDirs(s.bothPaths)
	addPathDirs := getSortedDirs(s.addPaths)
	_, addDirs, _ := diff(addPathDirs, bothPathDirs)
	addDirs = getSortedDirs(addDirs)

	for _, path := range addDirs {
		s.log(fmt.Sprintf("  make dir %s", path))

		err := s.dst.MakeDir(path, true)
		if err != nil {
			errors <- err
		}
	}
}

func (s *OneWay) writeFiles(errors chan<- error) {
	total := 0
	handled := 0
	logMain := func(msg string) {
		progress := 0.0
		if total > 0 {
			progress = 100.0 * float64(handled) / float64(total)
		}
		s.log(fmt.Sprintf("%.2f%% (%d/%d): %s", progress, handled, total, msg))
	}

	logMain("Writing files...")
	if len(s.addPaths) == 0 {
		logMain("Nothing to write")
		return
	}

	preparedFilePaths := []string{}
	for _, path := range s.addPaths {
		if res, exists := s.srcResources[path]; exists && !res.IsDir {
			preparedFilePaths = append(preparedFilePaths, path)
		}
	}
	sort.Slice(preparedFilePaths, func(i, j int) bool {
		return preparedFilePaths[i] < preparedFilePaths[j]
	})
	total = len(preparedFilePaths)

	paths := make(chan string)
	group := sync.WaitGroup{}

	thread := func(id uint) {
		curPath := "-"
		logThread := func(msg string) {
			logMain(fmt.Sprintf("[wthread %d] '%s': %s", id, curPath, msg))
		}
		logThread("Thread started")

		for {
			select {
			case path, ok := <-paths:
				if !ok {
					logThread("Thread exited")
					group.Done()
					return
				}
				curPath = path
				res := s.srcResources[path]

				var writeErr error = nil
				for i := uint(1); i <= s.opt.WriteRetry; i++ {
					logThread(fmt.Sprintf("Try %d / %d", i, s.opt.WriteRetry))
					writeErr = s.writeFile(path, res, logThread)
					if writeErr == nil {
						break
					}
					logThread(fmt.Sprintf("Try %d / %d ERR: '%v'", i, s.opt.WriteRetry, writeErr))
					time.Sleep(s.opt.WriteRetryDelay)
				}
				if writeErr != nil {
					logThread(fmt.Sprintf("ERROR '%v'", writeErr))
					errors <- writeErr
				} else {
					logThread("Complete")
				}
				handled++
				curPath = "-"
			}
		}
	}

	for i := uint(0); i < s.opt.WriteThreads; i++ {
		group.Add(1)
		go thread(i)
	}
	for _, path := range preparedFilePaths {
		paths <- path
	}
	close(paths)

	group.Wait()

	logMain("Write files complete")
}

func (s *OneWay) isSingleThreadWriteNeeded(res client.Resource) bool {
	if s.opt.SingleThreadedFileSize <= 0 {
		return false
	}
	return res.Size > s.opt.SingleThreadedFileSize
}

func (s *OneWay) writeFile(path string, res client.Resource, logFn func(string)) error {
	isSingleWriteLocked := true
	s.signleWriteMutex.Lock()

	if !s.isSingleThreadWriteNeeded(res) {
		isSingleWriteLocked = false
		s.signleWriteMutex.Unlock()
	} else {
		logFn("Single-thread write begin..")
	}

	unlockIfNeeded := func() {
		if isSingleWriteLocked {
			isSingleWriteLocked = false

			logFn("Single-thread write end")
			s.signleWriteMutex.Unlock()
		}
	}

	uploadPath := s.getUploadPath(path, res, s.opt.IndirectUpload)
	logFn(fmt.Sprintf("Uploading to '%s'", uploadPath))

	srcReader, err := s.src.ReadFile(path)
	if err != nil {
		unlockIfNeeded()
		return err
	}

	logReader := func(r *util.Reader) {
		logFn(fmt.Sprintf(
			"%.2f%% (%s / %s)",
			100*r.GetProgress(),
			util.FormatBytes(r.GetBytesRead()),
			util.FormatBytes(r.GetBytesTotal()),
		))
	}
	readerLogInterval := 2 * time.Second
	readerLogLastTime := time.Now()

	reader := util.NewRead(srcReader, res.Size)
	reader.OnProgress = func(r *util.Reader) {
		if time.Now().Sub(readerLogLastTime) >= readerLogInterval {
			readerLogLastTime = time.Now()
			logReader(r)
		}
	}
	reader.OnComplete = func(r *util.Reader) {
		logReader(r)
		unlockIfNeeded()
	}

	err = s.dst.WriteFile(uploadPath, reader, res.Size)
	time.Sleep(time.Second)

	unlockIfNeeded()
	reader.Close()

	logFn(fmt.Sprintf("readProgress IsComplete %t", reader.IsComplete()))
	logFn(fmt.Sprintf("err == nil: %t", err == nil))
	logFn(fmt.Sprintf("err %v, %#v", err, err))
	logFn(fmt.Sprintf("isErrEOF %t", isErrEOF(err)))

	if err != nil && !isErrEOF(err) {
		return err
	}

	logFn(fmt.Sprintf("Read bytes: %d", reader.GetBytesRead()))
	logFn(fmt.Sprintf("Read md5: %s", reader.GetHashMd5()))
	logFn(fmt.Sprintf("Read sha256: %s", reader.GetHashSha256()))

	logFn(fmt.Sprintf("Checking %s", uploadPath))
	err = s.checkWritten(uploadPath, res, reader, logFn)
	if err != nil {
		return err
	}

	if path != uploadPath {
		logFn(fmt.Sprintf("Moving %s", uploadPath))
		err = s.dst.MoveFile(uploadPath, path)
		if err != nil {
			return err
		}
	}

	return nil
}

func (s *OneWay) checkWritten(
	path string,
	res client.Resource,
	r *util.Reader,
	logFn func(string),
) (err error) {
	if !r.IsComplete() {
		return fmt.Errorf(
			"File not written. Stopped at %d of %d (%s / %s)",
			r.GetBytesRead(),
			r.GetBytesTotal(),
			util.FormatBytes(r.GetBytesRead()),
			util.FormatBytes(r.GetBytesTotal()),
		)
	}
	timeout := s.opt.WriteCheckTimeout
	timeStart := time.Now()
	for time.Now().Sub(timeStart) < timeout {
		logFn(fmt.Sprintf(
			"Checking (%s / %s) '%s'",
			time.Now().Sub(timeStart).String(),
			timeout.String(),
			path,
		))
		written, isExist, resErr := s.dst.GetResource(path)
		err = resErr
		if err == nil && isExist {
			err = s.checkWrittenRes(path, res, written, r, logFn)
			if err == nil {
				return
			}
		}
		time.Sleep(s.opt.WriteCheckDelay)
	}
	if err != nil {
		return err
	}
	return fmt.Errorf("File written but not found atfer timeout %s", timeout.String())
}

func (s *OneWay) checkWrittenRes(
	path string,
	src, written client.Resource,
	r *util.Reader,
	logFn func(string),
) (err error) {
	if written.HashSha256 != "" {
		if written.HashSha256 == r.GetHashSha256() {
			logFn("Check OK: SHA256 strict matched")
			return nil
		}
		logFn("Check FAIL: SHA256 not matched")
		return fmt.Errorf(
			"Written SHA256 not matched (%s -> %s), %s",
			r.GetHashSha256(),
			written.HashSha256,
			path,
		)
	}
	if written.HashMd5 != "" {
		if written.HashMd5 == r.GetHashMd5() {
			logFn("Check OK: MD5 strict matched")
			return nil
		}
		logFn("Check FAIL: MD5 not matched")
		return fmt.Errorf(
			"Written MD5 not matched (%s -> %s), %s",
			r.GetHashMd5(),
			written.HashMd5,
			path,
		)
	}
	if written.MatchAnyHash(r.GetHashSha256()) {
		logFn("Check OK: SHA256 matched")
		return nil
	}
	if written.MatchAnyHash(r.GetHashMd5()) {
		logFn("Check OK: MD5 matched")
		return nil
	}
	if written.Size == src.Size && src.Size == r.GetBytesRead() {
		logFn("Check OK: size matched")
		return nil
	}
	logFn("Check FAIL: size not matched")
	return fmt.Errorf(
		"Written size not matched (%d -> %d), %s",
		src.Size,
		written.Size,
		path,
	)
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

func diff(from, to []string) (both, add, del []string) {
	both = []string{}
	add = []string{}
	del = []string{}
	fromDict := map[string]bool{}
	for _, path := range from {
		fromDict[path] = true
	}
	toDict := map[string]bool{}
	for _, path := range to {
		toDict[path] = true
	}
	for _, path := range from {
		if _, exists := toDict[path]; exists {
			both = append(both, path)
		} else {
			add = append(add, path)
		}
	}
	for _, path := range to {
		if _, exists := fromDict[path]; !exists {
			del = append(del, path)
		}
	}
	return
}

func getSortedDirs(paths []string) []string {
	re := regexp.MustCompile("^.*/")
	dict := map[string]string{}
	for _, p := range paths {
		dir := re.FindString(p)
		if dir != "" {
			dict[dir] = dir
		}
	}
	sorted := []string{}
	for p := range dict {
		sorted = append(sorted, p)
	}
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i] < sorted[j]
	})
	return sorted
}

func isErrEOF(err error) bool {
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
