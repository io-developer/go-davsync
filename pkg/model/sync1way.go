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
	IgnoreExisting bool
	IndirectUpload bool
	AllowDelete    bool
	WriteThreads   uint
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
	s.logDiff()

	s.makeDirs(errors)
	s.writeFiles(errors)
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
	log.Println()
	log.Println("Source paths:")
	for _, path := range s.srcPaths {
		log.Println(path)
	}
	log.Println()

	log.Println()
	log.Println("Destination paths:")
	for _, path := range s.dstPaths {
		log.Println(path)
	}
	log.Println()
}

func (s *Sync1Way) diff() {
	from := []string{}
	for path := range s.srcResources {
		from = append(from, path)
	}
	to := []string{}
	for path := range s.dstNodes {
		to = append(to, path)
	}
	s.bothPaths, s.addPaths, s.delPaths = diff(from, to)
}

func (s *Sync1Way) logDiff() {
	for _, path := range s.bothPaths {
		log.Println("BOTH", path)
	}
	for _, path := range s.addPaths {
		log.Println(" ADD", path)
	}
	for _, path := range s.delPaths {
		log.Println(" DEL", path)
	}
}

func (s *Sync1Way) makeDirs(errors chan<- error) {
	log.Println("Making dirs...")

	bothPathDirs := getSortedDirs(s.bothPaths)
	addPathDirs := getSortedDirs(s.addPaths)
	_, addDirs, _ := diff(addPathDirs, bothPathDirs)
	addDirs = getSortedDirs(addDirs)

	for _, path := range addDirs {
		log.Println("  make dir", path)
		err := s.dst.MakeDir(path, true)
		if err != nil {
			errors <- err
		}
	}
}

func (s *Sync1Way) writeFiles(errors chan<- error) {
	log.Println("Writing files...")
	if len(s.addPaths) == 0 {
		log.Println("  nothing to write")
		return
	}

	sortedPaths := make([]string, len(s.addPaths))
	copy(sortedPaths, s.addPaths)
	sort.Slice(sortedPaths, func(i, j int) bool {
		return sortedPaths[i] < sortedPaths[j]
	})

	paths := make(chan string)
	group := sync.WaitGroup{}

	thread := func(id uint) {
		log.Printf("%d Write thread started\n", id)
		for {
			select {
			case path, ok := <-paths:
				if !ok {
					log.Printf("%d Write thread exited\n", id)
					group.Done()
					return
				}
				if res, exists := s.srcResources[path]; exists && !res.IsDir {
					log.Printf("%d Write thread writing '%s'\n", id, path)
					err := s.writeFile(path, res)
					if err != nil {
						log.Printf("%d Write thread error '%v'\n", id, err)
						errors <- err
					}
				}
			}
		}
	}

	for i := uint(0); i < s.opt.WriteThreads; i++ {
		group.Add(1)
		go thread(i)
	}
	for _, path := range sortedPaths {
		paths <- path
	}
	close(paths)

	group.Wait()
}

func (s *Sync1Way) writeFile(path string, res client.Resource) error {
	log.Println()
	log.Println("writing", path)

	uploadPath := path
	if s.opt.IndirectUpload {
		uploadPath = s.getUploadPath(path)
		log.Println("  indirect upload to ", uploadPath)
	}
	reader, err := s.src.ReadFile(path)
	if err != nil {
		return err
	}
	readProgress := NewReadProgress(reader, res.Size)
	err = s.dst.WriteFile(uploadPath, readProgress)
	if err != nil {
		reader.Close()
		return err
	}
	if path != uploadPath {
		log.Printf("  moving %s -> %s\n", uploadPath, path)
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
		fmt.Println(p)
		dir := re.FindString(p)
		fmt.Println("matched dir", dir)

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
