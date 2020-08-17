package model

import "log"

type Sync1WayOpt struct {
	IgnoreExisting bool
	AllowDelete    bool
}

type Sync1Way struct {
	src Client
	dst Client
	opt Sync1WayOpt

	// sync-time data
	srcPaths []string
	srcNodes map[string]Node

	dstPaths []string
	dstNodes map[string]Node

	bothPaths []string
	addPaths  []string
	delPaths  []string
}

func NewSync1Way(src, dst Client, opt Sync1WayOpt) *Sync1Way {
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

	s.srcPaths, s.srcNodes, err = s.src.ReadTree()
	if err != nil {
		return err
	}
	logTree(s.srcPaths, s.srcNodes)

	s.dstPaths, s.dstNodes, err = s.dst.ReadTree()
	if err != nil {
		return err
	}
	logTree(s.dstPaths, s.dstNodes)

	return err
}

func logTree(paths []string, nodes map[string]Node) {
	for _, path := range paths {
		log.Println(path)
	}
	for path, node := range nodes {
		log.Printf("\n%s\n%#v\n\n", path, node)
	}
}

func (s *Sync1Way) diff() {
	s.bothPaths, s.addPaths, s.delPaths = NodeComparePaths(s.srcNodes, s.dstNodes)
	for _, path := range s.bothPaths {
		log.Println("BOTH", path)
	}
	for _, path := range s.addPaths {
		log.Println("ADD", path)
	}
	for _, path := range s.delPaths {
		log.Println("DEL", path)
	}
}

func (s *Sync1Way) makeDirs() error {
	for _, path := range s.addPaths {
		node := s.srcNodes[path]
		if node.IsDir {
			log.Println("TRY ADD DIR", path)
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
		node := s.srcNodes[path]
		if !node.IsDir {
			log.Println("TRY WRITE FILE", path)

			reader, err := s.src.ReadFile(path)
			if err != nil {
				return err
			}
			readProgress := NewReadProgress(reader, node.Size)
			err = s.dst.WriteFile(path, readProgress)
			if err != nil {
				return err
			}
		}
	}
	return nil
}
