package main

import (
	"fmt"

	"github.com/io-developer/go-davsync/pkg/client"
	"github.com/io-developer/go-davsync/pkg/client/local"
	"github.com/io-developer/go-davsync/pkg/client/webdav"
	"github.com/io-developer/go-davsync/pkg/client/yadisk"
	"github.com/io-developer/go-davsync/pkg/client/yadiskrest"
	"github.com/io-developer/go-davsync/pkg/log"
	"github.com/io-developer/go-davsync/pkg/synchronizer"
)

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

// Client types
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

// Sync types
const (
	SyncTypeOneWay = SyncType("OneWay")
)

func main() {
	log.DefaultLogger.SetLevel(log.InfoLevel)

	args, err := parseArgs()
	if err != nil {
		log.Fatal("Error at cli args parsing", err)
	}
	log.Debugf("CLI ARGS:\n%#v\n\n", args)

	input, err := createClient(args.inputConfig)
	if err != nil {
		log.Fatal("Input client creation error", err)
	}
	output, err := createClient(args.outputConfig)
	if err != nil {
		log.Fatal("Output client creation error", err)
	}
	err = sync(input, output, args.syncConfig)
	if err != nil {
		log.Fatal("Sync error", err)
	}

	log.Info("\n\nDone.")
}

func createClient(conf ClientConfig) (c client.Client, err error) {
	switch conf.Type {
	case ClientTypeLocal:
		c = local.NewClient(conf.LocalOptions)
		return
	case ClientTypeWebdav:
		c = webdav.NewClient(conf.WebdavOptions)
		return
	case ClientTypeYadiskRest:
		c = yadiskrest.NewClient(conf.YadiskRestOptions)
		return
	case ClientTypeYadisk:
		c = yadisk.NewClient(
			webdav.NewClient(conf.WebdavOptions),
			yadiskrest.NewClient(conf.YadiskRestOptions),
		)
		return
	}
	err = fmt.Errorf("Unexpected client type '%s'", conf.Type)
	return
}

func sync(input, output client.Client, conf SyncConfig) error {
	if conf.Type == SyncTypeOneWay {
		return syncOnewWay(input, output, conf)
	}
	return fmt.Errorf("Unexpected sync-type '%s'", string(conf.Type))
}

func syncOnewWay(input, output client.Client, conf SyncConfig) error {
	log.Debug("Sync OneWay start..")

	s := synchronizer.NewOneWay(input, output, conf.OneWay)

	errors := make(chan error)
	go func(errors <-chan error) {
		log.Info("Listening for errors...")
		for {
			select {
			case err, ok := <-errors:
				if !ok {
					return
				}
				log.Error("!!! ERROR", err)
			}
		}
	}(errors)

	s.Sync(errors)
	close(errors)

	log.Debug("Sync OneWay end")

	return nil
}
