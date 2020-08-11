package main

import (
	"encoding/json"
	"encoding/xml"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"strings"
	"time"
	//	"github.com/studio-b12/gowebdav"
)

type DavOpt struct {
	BaseURI string
	User    string
	Pass    string
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
	XMLName        xml.Name  `xml:"DAV: response"`
	Href           string    `xml:"href"`
	Status         string    `xml:"propstat>status"`
	CreationDate   DavTime   `xml:"propstat>prop>creationdate"`
	LastModifyDate DavTime   `xml:"propstat>prop>getlastmodified"`
	DisplayName    string    `xml:"propstat>prop>displayname"`
	IsCollection   *struct{} `xml:"propstat>prop>resourcetype>collection"`
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

/*
type PropStat struct {
	XMLName xml.Name `xml:"DAV: propstat"`
	Status  string   `xml:"status"`
	//	Prop    Prop     `xml:"prop"`
	CreationDate   string `xml:"prop>creationdate"`
	LastModifyDate string `xml:"prop>getlastmodified"`
	DisplayName    string `xml:"prop>displayname"`
}
*/
/*
type Prop struct {
	XMLName        xml.Name `xml:"DAV: prop"`
	CreationDate   string   `xml:"DAV: creationdate"`
	LastModifyDate string   `xml:"DAV: getlastmodified"`
	DisplayName    string   `xml:"DAV: displayname"`
	ResourceType   string   `xml:"DAV: resourcetype"`
}
*/

func main() {
	opt, err := readOptFile("./.davsync")
	if err != nil {
		log.Fatalln("readOptFile err", err)
	}

	log.Printf("opt: %#v\n", opt)

	/*
		c := gowebdav.NewClient(baseURI, login, pass)

		err := c.Connect()
		if err != nil {
			log.Fatalln("Connect error", err)
		}

		infos, err := c.ReadDir("/backup/")
		if err != nil {
			log.Fatalln("ReadDir error", err)
		}

		log.Println("ReadDir infos", infos)
	*/

	path := "//backup/conf.pwd/"
	reqUri := fmt.Sprintf("%s/%s", strings.TrimRight(opt.BaseURI, "/"), strings.Trim(path, "/"))
	reqMethod := "PROPFIND"
	/*
		reqBody := `<d:propfind xmlns:d='DAV:'>
				<d:prop>
					<d:displayname/>
					<d:resourcetype/>
					<d:getcontentlength/>
					<d:getcontenttype/>
					<d:getetag/>
					<d:getlastmodified/>
				</d:prop>
			</d:propfind>`
	*/
	reqBody := `<d:propfind xmlns:d='DAV:'>
			<d:allprop/>
		</d:propfind>`
	req, err := http.NewRequest(reqMethod, reqUri, strings.NewReader(reqBody))
	if err != nil {
		log.Fatalln("NewRequest err", err)
	}

	req.SetBasicAuth(opt.User, opt.Pass)
	req.Header.Add("Content-Type", "application/xml;charset=UTF-8")
	req.Header.Add("Accept", "application/xml,text/xml")
	req.Header.Add("Accept-Charset", "utf-8")
	req.Header.Add("Accept-Encoding", "")
	req.Header.Add("Depth", "1")

	httpClient := http.Client{}
	resp, err := httpClient.Do(req)
	if err != nil {
		log.Fatalln("httpClient Do err", err)
	}

	log.Println("resp.StatusCode", resp.StatusCode)
	log.Println("resp.Body", resp.Body)

	respBytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Fatalln("Body Read err", err)
	}

	log.Println("respBytes", string(respBytes))

	propfindMulti := &PropfindMultistatus{}
	err = xml.Unmarshal(respBytes, propfindMulti)
	if err != nil {
		log.Fatalln("xml.Unmarshal err", err)
	}

	log.Printf("propfindMulti: %#v\n", propfindMulti)
}
