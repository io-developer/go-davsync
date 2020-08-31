package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"time"

	"github.com/io-developer/go-davsync/pkg/client"
	"github.com/io-developer/go-davsync/pkg/client/fs"
	"github.com/io-developer/go-davsync/pkg/client/webdav"
	"github.com/io-developer/go-davsync/pkg/client/yadiskrest"
	"github.com/io-developer/go-davsync/pkg/synchronizer"
)

type Args struct {
	input           string
	inputConfig     Config
	inputConfigFile string

	output           string
	outputConfig     Config
	outputConfigFile string
}

type Config struct {
	BaseDir    string
	Type       ClientType
	Webdav     webdav.Options
	YadiskRest yadiskrest.Options
}

type ClientType string

const (
	ClientTypeLocal      = ClientType("Local")
	ClientTypeWebdav     = ClientType("Webdav")
	ClientTypeYadiskRest = ClientType("YadiskRest")
	ClientTypeYadisk     = ClientType("Yadisk")
)

func parseArgs() (args Args, err error) {
	flag.StringVar(&args.input, "i", "./", "Input directory path. Example: /tmp/test")
	flag.StringVar(&args.inputConfigFile, "iconf", "", "JSON secrets for source client")

	flag.StringVar(&args.output, "o", "/", "Output directory path. Example: /test")
	flag.StringVar(&args.outputConfigFile, "oconf", ".davsync", "JSON secrets for destination client")

	flag.Parse()

	args.inputConfig, err = parseConfig(args.inputConfigFile, args.input)
	if err != nil {
		return
	}
	args.outputConfig, err = parseConfig(args.outputConfigFile, args.output)
	if err != nil {
		return
	}
	return
}

func parseConfig(path string, defBaseDir string) (conf Config, err error) {
	if path != "" {
		var bytes []byte
		bytes, err = ioutil.ReadFile(path)
		log.Println("parseConfig bytes", path, string(bytes))
		if err != nil {
			return
		}
		err = json.Unmarshal(bytes, &conf)
		if err != nil {
			return
		}
	}
	if conf.BaseDir == "" {
		conf.BaseDir = defBaseDir
	}
	if conf.Type == ClientType("") {
		conf.Type = ClientTypeLocal
	}
	if conf.Webdav.BaseDir == "" {
		conf.Webdav.BaseDir = conf.BaseDir
	}
	if conf.YadiskRest.BaseDir == "" {
		conf.YadiskRest.BaseDir = conf.BaseDir
	}
	if conf.YadiskRest.ApiUri == "" {
		conf.YadiskRest.ApiUri = "https://cloud-api.yandex.net/v1/disk"
	}
	return
}

func createClient(conf Config) (client.Client, error) {
	if conf.Type == ClientTypeLocal {
		return fs.NewClient(conf.BaseDir), nil
	}
	if conf.Type == ClientTypeWebdav {
		return webdav.NewClient(conf.Webdav), nil
	}
	if conf.Type == ClientTypeYadiskRest {
		return yadiskrest.NewClient(conf.YadiskRest), nil
	}
	if conf.Type == ClientTypeYadisk {
		rest := yadiskrest.NewClient(conf.YadiskRest)
		dav := webdav.NewClient(conf.Webdav)
		dav.SetTree(rest)
		return dav, nil
	}
	return nil, fmt.Errorf("Unexpected client type '%s'", conf.Type)
}

func main() {
	args, err := parseArgs()
	if err != nil {
		log.Fatalln("Error at paring cli args", err)
	}
	log.Printf("CLI ARGS:\n%#v\n\n", args)

	input, err := createClient(args.inputConfig)
	if err != nil {
		log.Fatalln("Input client creating error", err)
	}
	output, err := createClient(args.outputConfig)
	if err != nil {
		log.Fatalln("Output client creating error", err)
	}

	sync := synchronizer.NewOneWay(input, output, synchronizer.OneWayOpt{
		IndirectUpload:         true,
		IgnoreExisting:         true,
		AllowDelete:            false,
		SingleThreadedFileSize: 64 * 1024 * 1024,
		WriteThreads:           8,
		WriteRetry:             2,
		WriteRetryDelay:        30 * time.Second,
		WriteCheckTimeout:      30 * time.Minute,
		WriteCheckDelay:        10 * time.Second,
	})

	errors := make(chan error)
	go listenErrors(errors)

	sync.Sync(errors)

	log.Println("\n\nDone.")
}

func listenErrors(errors <-chan error) {
	log.Println("Listening for errors...")
	for {
		select {
		case err, ok := <-errors:
			if !ok {
				return
			}
			log.Println("!!! ERROR", err)
		}
	}
}
