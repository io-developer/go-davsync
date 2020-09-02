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
	ThreadCount            uint
	AttemptMax             uint
	AttemptDelay           time.Duration
	UploadCheckTimeout     time.Duration
	UploadCheckDelay       time.Duration
}

type OneWay struct {
	input  client.Client
	output client.Client
	opt    OneWayOpt

	// sync-time data
	inputPaths     []string
	inputResources map[string]client.Resource

	outputPaths     []string
	outputResources map[string]client.Resource

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
		input:              input,
		output:             output,
		opt:                opt,
		signleThreadUpload: sync.Mutex{},
	}
}

func (s *OneWay) Sync(errors chan<- error) {
	s.readTrees(errors)
	s.logTrees()

	s.diff()
	s.makeDirs(errors)
	s.uploadFiles(errors)
	s.deleteFiles(errors)
}

func (s *OneWay) log(msg string) {
	log.Printf("Sync: %s\n", msg)
}

func (s *OneWay) readTrees(errors chan<- error) {
	group := sync.WaitGroup{}
	group.Add(2)
	go func() {
		var err error
		s.inputPaths, s.inputResources, err = s.input.ReadTree()
		if err != nil {
			errors <- err
		}
		group.Done()
	}()
	go func() {
		var err error
		s.outputPaths, s.outputResources, err = s.output.ReadTree()
		if err != nil {
			errors <- err
		}
		group.Done()
	}()
	group.Wait()
}

func (s *OneWay) logTrees() {
	s.log("")
	s.log("Input existing paths:")
	for _, path := range s.inputPaths {
		s.log(path)
		//	res := s.srcResources[path]
		//	s.log(fmt.Sprintf("ModTime: %s", res.ModTime.Format("2006-01-02 15:04:05 -0700")))
	}
	s.log("")

	s.log("")
	s.log("Output existing paths:")
	for _, path := range s.outputPaths {
		s.log(path)
		//	res := s.dstNodes[path]
		//	s.log(fmt.Sprintf("ModTime: %s", res.ModTime.Format("2006-01-02 15:04:05 -0700")))
	}
	s.log("")
}

func (s *OneWay) diff() {
	s.log("Calculating input/output path diff...")

	from := []string{}
	for path := range s.inputResources {
		from = append(from, path)
	}
	to := []string{}
	for path := range s.outputResources {
		to = append(to, path)
	}
	s.bothPaths, s.addPaths, s.delPaths = diff(from, to)

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

	bothPathDirs := getSortedDirs(s.bothPaths)
	addPathDirs := getSortedDirs(s.addPaths)
	_, addDirs, _ := diff(addPathDirs, bothPathDirs)
	addDirs = getSortedDirs(addDirs)

	for _, path := range addDirs {
		s.log(fmt.Sprintf("  make dir %s", path))

		err := s.output.MakeDir(path, true)
		if err != nil {
			errors <- err
		}
	}
}

func (s *OneWay) uploadFiles(errors chan<- error) {
	total := 0
	handled := 0
	logMain := func(msg string) {
		progress := 0.0
		if total > 0 {
			progress = 100.0 * float64(handled) / float64(total)
		}
		s.log(fmt.Sprintf("U %.2f%% (%d/%d): %s", progress, handled, total, msg))
	}

	logMain("Uploading files...")
	if len(s.addPaths) == 0 {
		logMain("Nothing to upload")
		return
	}

	preparedFilePaths := []string{}
	for _, path := range s.addPaths {
		if res, exists := s.inputResources[path]; exists && !res.IsDir {
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
			logMain(fmt.Sprintf("[uthread %d] '%s': %s", id, curPath, msg))
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
				res := s.inputResources[path]

				var uploadErr error = nil
				for i := uint(1); i <= s.opt.AttemptMax; i++ {
					logThread(fmt.Sprintf("Try %d / %d", i, s.opt.AttemptMax))
					uploadErr = s.uploadFile(path, res, logThread)
					if uploadErr == nil {
						break
					}
					logThread(fmt.Sprintf("Try %d / %d ERR: '%v'", i, s.opt.AttemptMax, uploadErr))
					time.Sleep(s.opt.AttemptDelay)
				}
				if uploadErr != nil {
					logThread(fmt.Sprintf("ERROR '%v'", uploadErr))
					errors <- uploadErr
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
	for _, path := range preparedFilePaths {
		paths <- path
	}
	close(paths)

	group.Wait()

	logMain("Upload files complete")
}

func (s *OneWay) isSingleThreadUploadNeeded(res client.Resource) bool {
	if s.opt.SingleThreadedFileSize <= 0 {
		return false
	}
	return res.Size > s.opt.SingleThreadedFileSize
}

func (s *OneWay) uploadFile(path string, res client.Resource, logFn func(string)) error {
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

	srcReader, err := s.input.ReadFile(path)
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

	err = s.output.WriteFile(uploadPath, reader, res.Size)
	time.Sleep(time.Second)

	unlockIfNeeded()
	reader.Close()

	logFn(fmt.Sprintf("Reader IsComplete %t", reader.IsComplete()))
	logFn(fmt.Sprintf("Reader err isErrEOF %t", isErrEOF(err)))

	if err != nil && !isErrEOF(err) {
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

func (s *OneWay) getUploadPath(path string, res client.Resource, indirect bool) string {
	if !indirect {
		return path
	}
	sign := fmt.Sprintf("%s:%d", path, res.Size)
	h := crypto.SHA256.New()
	h.Write([]byte(sign))
	return fmt.Sprintf(s.opt.UploadPathFormat, h.Sum(nil))
}

func (s *OneWay) deleteFiles(errors chan<- error) {
	total := 0
	handled := 0
	logMain := func(msg string) {
		progress := 0.0
		if total > 0 {
			progress = 100.0 * float64(handled) / float64(total)
		}
		s.log(fmt.Sprintf("D %.2f%% (%d/%d): %s", progress, handled, total, msg))
	}

	logMain("Deleting files...")
	if !s.opt.AllowDelete {
		logMain("Deleting disabled. Skipping..")
		return
	}
	if len(s.delPaths) == 0 {
		logMain("Nothing to delete")
		return
	}

	preparedFilePaths := []string{}
	for _, path := range s.delPaths {
		if res, exists := s.outputResources[path]; exists && !res.IsDir {
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
			logMain(fmt.Sprintf("[dthread %d] '%s': %s", id, curPath, msg))
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
				var delErr error = nil
				for i := uint(1); i <= s.opt.AttemptMax; i++ {
					logThread(fmt.Sprintf("Try %d / %d", i, s.opt.AttemptMax))
					delErr = s.output.DeleteFile(path)
					if delErr == nil {
						break
					}
					logThread(fmt.Sprintf("Try %d / %d ERR: '%v'", i, s.opt.AttemptMax, delErr))
					time.Sleep(s.opt.AttemptDelay)
				}
				if delErr != nil {
					logThread(fmt.Sprintf("ERROR '%v'", delErr))
					errors <- delErr
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
	for _, path := range preparedFilePaths {
		paths <- path
	}
	close(paths)

	group.Wait()

	logMain("Delete files complete")
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
