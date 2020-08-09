package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"strings"
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
}
