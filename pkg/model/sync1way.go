package model

import (
	"fmt"
	"log"
	"regexp"
	"sort"
	"sync"
	"time"

	"github.com/io-developer/go-davsync/pkg/client"
)

type Sync1WayOpt struct {
	IgnoreExisting         bool
	IndirectUpload         bool
	AllowDelete            bool
	WriteThreads           uint
	WriteRetry             uint
	WriteRetryWait         time.Duration
	SingleThreadedFileSize int64
}

type Sync1Way struct {
	src client.Client
	dst client.Client
	opt Sync1WayOpt

	// sync-time data
	srcPaths     []string
	srcResources map[string]client.Resource

	dstPaths []string
	dstNodes map[string]client.Resource

	bothPaths []string
	addPaths  []string
	delPaths  []string
}

func NewSync1Way(src, dst client.Client, opt Sync1WayOpt) *Sync1Way {
	if opt.WriteThreads < 1 {
		opt.WriteThreads = 1
	}
	if opt.WriteRetry < 1 {
		opt.WriteRetry = 1
	}
	return &Sync1Way{
		src: src,
		dst: dst,
		opt: opt,
	}
}

func (s *Sync1Way) Sync(errors chan<- error) {
	s.readTrees(errors)
	s.logTrees()

	s.diff()
	s.makeDirs(errors)
	s.writeFiles(errors)
}

func (s *Sync1Way) log(msg string) {
	log.Printf("Sync: %s\n", msg)
}

func (s *Sync1Way) readTrees(errors chan<- error) {
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

func (s *Sync1Way) logTrees() {
	s.log("")
	s.log("Source paths:")
	for _, path := range s.srcPaths {
		s.log(path)
	}
	s.log("")

	s.log("")
	s.log("Destination paths:")
	for _, path := range s.dstPaths {
		s.log(path)
	}
	s.log("")
}

func (s *Sync1Way) diff() {
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

func (s *Sync1Way) makeDirs(errors chan<- error) {
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

func (s *Sync1Way) writeFiles(errors chan<- error) {
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

	sthreadWriteMutex := sync.Mutex{}

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
				sthreadWriteMutex.Lock()
				curPath = path

				res := s.srcResources[path]
				isSingleThreaded := s.isSingleThreadWriteNeeded(res)
				if isSingleThreaded {
					logThread("Single-thread writting start..")
				} else {
					logThread("Multi-thread writting")
					sthreadWriteMutex.Unlock()
				}
				var writeErr error = nil
				for i := uint(1); i <= s.opt.WriteRetry; i++ {
					logThread(fmt.Sprintf("Try %d / %d", i, s.opt.WriteRetry))
					writeErr = s.writeFile(path, res, logThread)
					if writeErr == nil {
						break
					}
					logThread(fmt.Sprintf("Try %d / %d ERR: '%v'", i, s.opt.WriteRetry, writeErr))
					time.Sleep(s.opt.WriteRetryWait)
				}
				if isSingleThreaded {
					logThread("Single-thread writting complete")
					sthreadWriteMutex.Unlock()
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

func (s *Sync1Way) isSingleThreadWriteNeeded(res client.Resource) bool {
	if s.opt.SingleThreadedFileSize <= 0 {
		return false
	}
	return res.Size > s.opt.SingleThreadedFileSize
}

func (s *Sync1Way) writeFile(path string, res client.Resource, logFn func(string)) error {
	uploadPath := path
	if s.opt.IndirectUpload {
		uploadPath = s.getUploadPath(path)
		logFn(fmt.Sprintf("Indirect upload to '%s'", uploadPath))
	}
	reader, err := s.src.ReadFile(path)
	if err != nil {
		return err
	}
	readProgress := NewReadProgress(reader, res.Size)
	readProgress.SetLogFn(logFn)
	err = s.dst.WriteFile(uploadPath, readProgress, res.Size)
	if err != nil {
		reader.Close()
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

func (s *Sync1Way) getUploadPath(src string) string {
	return fmt.Sprintf("/ucam-%d.bin", time.Now().Local().UnixNano())
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
