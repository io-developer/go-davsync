package main

import (
	"encoding/json"
	"encoding/xml"
	"errors"
	"io/ioutil"
	"log"
	"time"

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

type PropfindMultistatus struct {
	XMLName   xml.Name   `xml:"DAV: multistatus"`
	Propfinds []Propfind `xml:"response"`
}

type Propfind struct {
	XMLName                xml.Name  `xml:"DAV: response"`
	Href                   string    `xml:"href"`
	Status                 string    `xml:"propstat>status"`
	CreationDate           DavTime   `xml:"propstat>prop>creationdate"`
	LastModified           DavTime   `xml:"propstat>prop>getlastmodified"`
	DisplayName            string    `xml:"propstat>prop>displayname"`
	Etag                   string    `xml:"propstat>prop>getetag"`
	ContentType            string    `xml:"propstat>prop>getcontenttype"`
	ContentLength          int64     `xml:"propstat>prop>getcontentlength"`
	ResourceTypeCollection *struct{} `xml:"propstat>prop>resourcetype>collection"`
}

func (c *Propfind) IsCollection() bool {
	return c.ResourceTypeCollection != nil
}

type DavTime struct {
	time.Time
}

func (t *DavTime) UnmarshalXML(d *xml.Decoder, start xml.StartElement) error {
	var elemVal string
	d.DecodeElement(&elemVal, &start)

	formats := []string{
		"Mon Jan 2 15:04:05 -0700 MST 2006",
		"Mon, 02 Jan 2006 15:04:05 MST",
		"2006-01-02T15:04:05Z",
	}
	for _, format := range formats {
		if time, err := time.Parse(format, elemVal); err == nil {
			*t = DavTime{time}
			return nil
		}
	}
	return errors.New("Cant parse time: " + elemVal)
}

func main() {
	fsClient := fs.NewClient("/home/iodev/projects/local/davsync/.dav-test-input")
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

	return
	opt, err := readOptFile("./.davsync")
	if err != nil {
		log.Fatalln("readOptFile err", err)
	}

	log.Printf("opt: %#v\n", opt)

	client := webdav.NewClient()
	client.BaseURI = opt.BaseURI
	client.AuthToken = opt.Token
	client.AuthTokenType = opt.TokenType
	client.AuthUser = opt.User
	client.AuthPass = opt.Pass

	propfindMulti, err := client.Propfind("/Загрузки/")
	if err != nil {
		log.Fatalln("Propfind err", err)
	}

	log.Printf("\n\npropfindMulti: %#v\n", propfindMulti)

	for _, resource := range propfindMulti.Propfinds {
		log.Println("Resource ", resource.Href)
		log.Println("  Status", resource.Status)
		log.Println("  IsCollection", resource.IsCollection())
		log.Println("  DisplayName", resource.DisplayName)
		log.Println("  CreationDate", resource.CreationDate.Format("2006-01-02 15:04:05 -0700"))
		log.Println("  LastModified", resource.LastModified.Format("2006-01-02 15:04:05 -0700"))
		log.Println("  ContentType", resource.ContentType)
		log.Println("  ContentLength", resource.ContentLength)
		log.Println("  Etag", resource.Etag)
	}
}
