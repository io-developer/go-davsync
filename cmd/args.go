package main

import (
	"encoding/json"
	"flag"
	"io/ioutil"
	"log"
	"time"

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

	// std sync flags
	threads  uint
	attempts uint
}

var defaultLocalOptions = local.Options{
	DirMode:  0755,
	FileMode: 0644,
}

var defaultWebdavOptions = webdav.Options{
	AuthTokenType: "OAuth",
}

var defaultYadiskRestOptions = yadiskrest.Options{
	ApiUri:          "https://cloud-api.yandex.net/v1/disk",
	AuthTokenType:   "OAuth",
	DeletePermanent: true,
}

var defaultInputClientConfig = ClientConfig{
	Type:              ClientTypeLocal,
	LocalOptions:      defaultLocalOptions,
	WebdavOptions:     defaultWebdavOptions,
	YadiskRestOptions: defaultYadiskRestOptions,
}

var defaultOutputClientConfig = ClientConfig{
	Type:              ClientTypeWebdav,
	LocalOptions:      defaultLocalOptions,
	WebdavOptions:     defaultWebdavOptions,
	YadiskRestOptions: defaultYadiskRestOptions,
}

var defaultSyncConfig = SyncConfig{
	Type: SyncTypeOneWay,
	OneWay: synchronizer.OneWayOpt{
		IndirectUpload:         true,
		IgnoreExisting:         true,
		AllowDelete:            false,            // append-only mode by default
		SingleThreadedFileSize: 64 * 1024 * 1024, // 64 MiB
		ThreadCount:            4,
		AttemptMax:             3,
		AttemptDelay:           30 * time.Second,
		UploadCheckDelay:       10 * time.Second,
		UploadCheckTimeout:     30 * time.Minute,
	},
}

func parseArgs() (args Args, err error) {
	flag.StringVar(&args.input, "i", "./", "Default input directory path. Example: /tmp/test")
	flag.StringVar(&args.inputConfigFile, "iconf", "", "Input client config JSON file")

	flag.StringVar(&args.output, "o", "/", "Default output directory path. Example: /test")
	flag.StringVar(&args.outputConfigFile, "oconf", ".davsync", "Output client config JSON file")

	flag.UintVar(&args.threads, "threads", 4, "Max threads")
	flag.UintVar(&args.attempts, "attempts", 3, "Max attempts")

	flag.StringVar(&args.sync, "sync", "OneWay", "Default sync type")
	flag.StringVar(&args.syncConfigFile, "syncConf", "", "Sync config JSON file")

	flag.Parse()

	args.inputConfig = defaultInputClientConfig
	err = parseClientConfig(args.inputConfigFile, &args.inputConfig, args.input)
	if err != nil {
		return
	}

	args.outputConfig = defaultOutputClientConfig
	err = parseClientConfig(args.outputConfigFile, &args.outputConfig, args.output)
	if err != nil {
		return
	}

	args.syncConfig = defaultSyncConfig
	err = parseSyncConfig(args.syncConfigFile, &args.syncConfig, args)
	if err != nil {
		return
	}
	return
}

func parseClientConfig(path string, outConf *ClientConfig, baseDir string) error {
	outConf.BaseDir = baseDir
	outConf.LocalOptions.BaseDir = baseDir
	outConf.WebdavOptions.BaseDir = baseDir
	outConf.YadiskRestOptions.BaseDir = baseDir

	if path != "" {
		var bytes []byte
		bytes, err := ioutil.ReadFile(path)
		log.Println("parseClientConfig bytes", path, string(bytes))
		if err != nil {
			return err
		}
		err = json.Unmarshal(bytes, outConf)
		if err != nil {
			return err
		}
	}
	return nil
}

func parseSyncConfig(path string, outConf *SyncConfig, args Args) error {
	outConf.OneWay.ThreadCount = args.threads
	outConf.OneWay.AttemptMax = args.attempts

	if path != "" {
		var bytes []byte
		bytes, err := ioutil.ReadFile(path)
		log.Println("parseSyncConfig bytes", path, string(bytes))
		if err != nil {
			return err
		}
		err = json.Unmarshal(bytes, outConf)
		if err != nil {
			return err
		}
	}
	return nil
}
