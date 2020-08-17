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
}

func NewSync1Way(src, dst Client, opt Sync1WayOpt) *Sync1Way {
	return &Sync1Way{
		src: src,
		dst: dst,
		opt: opt,
	}
}

func (s *Sync1Way) Sync() error {
	srcPaths, srcNodes, err := s.src.ReadTree()
	if err != nil {
		return err
	}
	logTree(srcPaths, srcNodes)

	dstPaths, dstNodes, err := s.dst.ReadTree()
	if err != nil {
		return err
	}
	logTree(dstPaths, dstNodes)

	bothPaths, addPaths, delPaths := NodeComparePaths(srcNodes, dstNodes)
	for _, path := range bothPaths {
		log.Println("BOTH", path)
	}
	for _, path := range addPaths {
		log.Println("ADD", path)
	}
	for _, path := range delPaths {
		log.Println("DEL", path)
	}

	for _, path := range addPaths {
		node := srcNodes[path]
		if node.IsDir {
			log.Println("TRY ADD DIR", path)
			err := s.dst.Mkdir(path, true)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func logTree(paths []string, nodes map[string]Node) {
	for _, path := range paths {
		log.Println(path)
	}
	for path, node := range nodes {
		log.Printf("\n%s\n%#v\n\n", path, node)
	}
}
