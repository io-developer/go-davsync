package main

import (
	"encoding/json"
	"io/ioutil"
	"log"
	"path/filepath"

	"github.com/io-developer/davsync/fs"
	"github.com/io-developer/davsync/webdav"
	//	"github.com/studio-b12/gowebdav"
)

type DavOpt struct {
	BaseURI   string
	Token     string
	TokenType string
	User      string
	Pass      string
}

func readOptFile(path string) (DavOpt, error) {
	bytes, err := ioutil.ReadFile(path)
	if err != nil {
		log.Fatal(err)
	}
	opt := DavOpt{}
	json.Unmarshal(bytes, &opt)

	return opt, nil
}

func main() {
	a := "/home/iodev"
	b := "/"
	log.Println("a", a)
	log.Println("b", b)
	log.Println("join", filepath.Join(a, b))

	fsClient := fs.NewClient(".dav-test-input")
	paths, nodes, err := fsClient.ReadTree()
	if err != nil {
		log.Fatalln("ReadTree err", err)
	}
	for _, path := range paths {
		log.Println(path)
	}
	for path, node := range nodes {
		log.Printf("%s\n%#v\n\n", path, node)
	}

	opt, err := readOptFile("./.davsync")
	if err != nil {
		log.Fatalln("readOptFile err", err)
	}

	log.Printf("opt: %#v\n", opt)

	davClient := webdav.NewClient()
	davClient.BaseURI = opt.BaseURI
	davClient.BasePath = "/Загрузки"
	davClient.AuthToken = opt.Token
	davClient.AuthTokenType = opt.TokenType
	davClient.AuthUser = opt.User
	davClient.AuthPass = opt.Pass

	propfindSome, err := davClient.PropfindSome("/", 1)
	if err != nil {
		log.Fatalln("Propfind err", err)
	}

	log.Printf("\n\nPropfindSome: %#v\n", propfindSome)
	for _, propfind := range propfindSome.Propfinds {
		logPropfind(propfind)
	}

	propfind, err := davClient.Propfind("/")
	if err != nil {
		log.Fatalln("Propfind err", err)
	}

	log.Printf("\n\nPropfind: %#v\n", propfindSome)
	logPropfind(propfind)

	paths, nodes, err = davClient.ReadTree()
	if err != nil {
		log.Fatalln("ReadTree err", err)
	}
	for _, path := range paths {
		log.Println(path)
	}
	for path, node := range nodes {
		log.Printf("\n%s\n%#v\n\n", path, node)
	}
}

func logPropfind(p webdav.Propfind) {
	log.Println("Resource ", p.Href)
	log.Println("  Status", p.Status)
	log.Println("  IsCollection", p.IsCollection())
	log.Println("  DisplayName", p.DisplayName)
	log.Println("  CreationDate", p.CreationDate.Format("2006-01-02 15:04:05 -0700"))
	log.Println("  LastModified", p.LastModified.Format("2006-01-02 15:04:05 -0700"))
	log.Println("  ContentType", p.ContentType)
	log.Println("  ContentLength", p.ContentLength)
	log.Println("  Etag", p.Etag)
}
