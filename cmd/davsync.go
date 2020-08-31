package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"time"

	"github.com/io-developer/go-davsync/pkg/client"
	"github.com/io-developer/go-davsync/pkg/client/local"
	"github.com/io-developer/go-davsync/pkg/client/webdav"
	"github.com/io-developer/go-davsync/pkg/client/yadiskrest"
	"github.com/io-developer/go-davsync/pkg/synchronizer"
)

// Args of cli
type Args struct {
	input           string
	inputConfig     ClientConfig
	inputConfigFile string

	output           string
	outputConfig     ClientConfig
	outputConfigFile string

	sync           string
	syncConfig     SyncConfig
	syncConfigFile string
}

// ClientConfig of input/output
type ClientConfig struct {
	BaseDir           string
	Type              ClientType
	LocalOptions      local.Options
	WebdavOptions     webdav.Options
	YadiskRestOptions yadiskrest.Options
}

// ClientType ..
type ClientType string

// Known types
const (
	ClientTypeLocal      = ClientType("Local")
	ClientTypeWebdav     = ClientType("Webdav")
	ClientTypeYadiskRest = ClientType("YadiskRest")
	ClientTypeYadisk     = ClientType("Yadisk")
)

// SyncConfig of sync
type SyncConfig struct {
	Type   SyncType
	OneWay synchronizer.OneWayOpt
}

// SyncType ..
type SyncType string

// Known types
const (
	SyncTypeOneWay = SyncType("OneWay")
)

func parseArgs() (args Args, err error) {
	flag.StringVar(&args.input, "i", "./", "Default input directory path. Example: /tmp/test")
	flag.StringVar(&args.inputConfigFile, "iconf", "", "Input client config JSON file")

	flag.StringVar(&args.output, "o", "/", "Default output directory path. Example: /test")
	flag.StringVar(&args.outputConfigFile, "oconf", ".davsync", "Output client config JSON file")

	flag.StringVar(&args.sync, "t", "OneWay", "Default sync type")
	flag.StringVar(&args.syncConfigFile, "tconf", "", "Sync config JSON file")

	flag.Parse()

	args.inputConfig, err = parseClientConfig(args.inputConfigFile, args.input)
	if err != nil {
		return
	}
	args.outputConfig, err = parseClientConfig(args.outputConfigFile, args.output)
	if err != nil {
		return
	}
	args.syncConfig, err = parseSyncConfig(args.syncConfigFile, SyncType(args.sync))
	if err != nil {
		return
	}
	return
}

func parseClientConfig(path string, defBaseDir string) (conf ClientConfig, err error) {
	if path != "" {
		var bytes []byte
		bytes, err = ioutil.ReadFile(path)
		log.Println("parseClientConfig bytes", path, string(bytes))
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

	if conf.LocalOptions.BaseDir == "" {
		conf.LocalOptions.BaseDir = conf.BaseDir
	}
	if conf.LocalOptions.FileMode == 0 {
		conf.LocalOptions.FileMode = 0644
	}
	if conf.LocalOptions.DirMode == 0 {
		conf.LocalOptions.DirMode = 0755
	}

	if conf.WebdavOptions.BaseDir == "" {
		conf.WebdavOptions.BaseDir = conf.BaseDir
	}

	if conf.YadiskRestOptions.BaseDir == "" {
		conf.YadiskRestOptions.BaseDir = conf.BaseDir
	}
	if conf.YadiskRestOptions.ApiUri == "" {
		conf.YadiskRestOptions.ApiUri = "https://cloud-api.yandex.net/v1/disk"
	}

	return
}

func createClient(conf ClientConfig) (client.Client, error) {
	if conf.Type == ClientTypeLocal {
		return local.NewClient(conf.LocalOptions), nil
	}
	if conf.Type == ClientTypeWebdav {
		return webdav.NewClient(conf.WebdavOptions), nil
	}
	if conf.Type == ClientTypeYadiskRest {
		return yadiskrest.NewClient(conf.YadiskRestOptions), nil
	}
	if conf.Type == ClientTypeYadisk {
		rest := yadiskrest.NewClient(conf.YadiskRestOptions)
		dav := webdav.NewClient(conf.WebdavOptions)
		dav.SetTree(rest)
		return dav, nil
	}
	return nil, fmt.Errorf("Unexpected client type '%s'", conf.Type)
}

func parseSyncConfig(path string, defType SyncType) (conf SyncConfig, err error) {
	if path != "" {
		var bytes []byte
		bytes, err = ioutil.ReadFile(path)
		log.Println("parseSyncConfig bytes", path, string(bytes))
		if err != nil {
			return
		}
		err = json.Unmarshal(bytes, &conf)
		if err != nil {
			return
		}
	} else {
		conf = SyncConfig{
			Type: defType,
			OneWay: synchronizer.OneWayOpt{
				IndirectUpload:         true,
				IgnoreExisting:         true,
				AllowDelete:            false,
				SingleThreadedFileSize: 64 * 1024 * 1024,
				WriteThreads:           4,
				WriteRetry:             2,
				WriteRetryDelay:        30 * time.Second,
				WriteCheckTimeout:      30 * time.Minute,
				WriteCheckDelay:        10 * time.Second,
			},
		}
	}
	if conf.Type == "" {
		conf.Type = defType
	}
	return
}

func sync(input, output client.Client, conf SyncConfig) error {
	if conf.Type == SyncTypeOneWay {
		return syncOnewWay(input, output, conf)
	}
	return fmt.Errorf("Unexpected sync-type '%s'", string(conf.Type))
}

func syncOnewWay(input, output client.Client, conf SyncConfig) error {
	log.Println("Sync OneWay start..")

	s := synchronizer.NewOneWay(input, output, conf.OneWay)

	errors := make(chan error)
	go func(errors <-chan error) {
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
	}(errors)

	s.Sync(errors)
	close(errors)

	log.Println("Sync OneWay end")

	return nil
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
	err = sync(input, output, args.syncConfig)
	if err != nil {
		log.Fatalln("Sync error", err)
	}

	log.Println("\n\nDone.")
}
