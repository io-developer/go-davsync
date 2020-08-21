package model

import (
	"fmt"
	"log"
	"time"

	"github.com/io-developer/go-davsync/pkg/client"
)

type Sync1WayOpt struct {
	IgnoreExisting bool
	IndirectUpload bool
	AllowDelete    bool
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
	return &Sync1Way{
		src: src,
		dst: dst,
		opt: opt,
	}
}

func (s *Sync1Way) Sync() error {
	err := s.readTrees()
	if err != nil {
		return err
	}
	s.diff()
	err = s.makeDirs()
	if err != nil {
		return err
	}
	err = s.writeFiles()
	if err != nil {
		return err
	}
	return nil
}

func (s *Sync1Way) readTrees() error {
	var err error

	s.srcPaths, s.srcResources, err = s.src.ReadTree()
	if err != nil {
		return err
	}
	logTree(s.srcPaths, s.srcResources)

	s.dstPaths, s.dstNodes, err = s.dst.ReadTree()
	if err != nil {
		return err
	}
	logTree(s.dstPaths, s.dstNodes)

	return err
}

func logTree(paths []string, nodes map[string]client.Resource) {
	for _, path := range paths {
		log.Println(path)
	}
	for path, node := range nodes {
		log.Printf("\n%s\n%#v\n\n", path, node)
	}
}

func (s *Sync1Way) diff() {
	s.bothPaths, s.addPaths, s.delPaths = compareNodes(s.srcResources, s.dstNodes)
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

func (s *Sync1Way) makeDirs() error {
	for _, path := range s.addPaths {
		node := s.srcResources[path]
		if node.IsDir {
			log.Println("MKDIR", path)
			err := s.dst.MakeDir(path, true)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func (s *Sync1Way) writeFiles() error {
	for _, path := range s.addPaths {
		res := s.srcResources[path]
		if !res.IsDir {
			err := s.writeFile(path, res)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func (s *Sync1Way) writeFile(path string, res client.Resource) error {
	log.Println()
	log.Println("WRITE FILE", path)

	uploadPath := path
	if s.opt.IndirectUpload {
		uploadPath = getUploadPath(path)
		log.Println("  INDIRECT UPLOAD ", uploadPath)
	}
	err := s.dst.MakeDirFor(uploadPath)
	if err != nil {
		return err
	}
	err = s.dst.MakeDirFor(path)
	if err != nil {
		return err
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
		log.Printf("  MOVING %s -> %s\n", uploadPath, path)
		err = s.dst.MoveFile(uploadPath, path)
		if err != nil {
			return err
		}
	}
	return nil
}

func getUploadPath(src string) string {
	return fmt.Sprintf("/ucam-%s.bin", time.Now().Format("20060102150405"))
}

func compareNodes(from, to map[string]client.Resource) (both, add, del []string) {
	both = []string{}
	add = []string{}
	del = []string{}
	for path := range from {
		if _, exists := to[path]; exists {
			both = append(both, path)
		} else {
			add = append(add, path)
		}
	}
	for path := range to {
		if _, exists := from[path]; !exists {
			del = append(del, path)
		}
	}
	return
}
