package webdav

import (
	"encoding/xml"
	"errors"
	"net/url"
	"time"
)

type PropfindSome struct {
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

func (p *Propfind) IsCollection() bool {
	return p.ResourceTypeCollection != nil
}

func (p *Propfind) GetHrefUnicode() string {
	if decoded, err := url.QueryUnescape(p.Href); err == nil {
		return decoded
	}
	return p.Href
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
