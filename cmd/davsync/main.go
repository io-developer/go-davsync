package main

import (
	"fmt"
	"log"

	"github.com/io-developer/go-davsync/pkg/client"
	"github.com/io-developer/go-davsync/pkg/client/local"
	"github.com/io-developer/go-davsync/pkg/client/webdav"
	"github.com/io-developer/go-davsync/pkg/client/yadiskrest"
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
	args, err := parseArgs()
	if err != nil {
		log.Fatalln("Error at cli args parsing", err)
	}
	log.Printf("CLI ARGS:\n%#v\n\n", args)

	input, err := createClient(args.inputConfig)
	if err != nil {
		log.Fatalln("Input client creation error", err)
	}
	output, err := createClient(args.outputConfig)
	if err != nil {
		log.Fatalln("Output client creation error", err)
	}
	err = sync(input, output, args.syncConfig)
	if err != nil {
		log.Fatalln("Sync error", err)
	}

	log.Println("\n\nDone.")
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
		//	rest := yadiskrest.NewClient(conf.YadiskRestOptions)
		dav := webdav.NewClient(conf.WebdavOptions)
		//	dav.SetTree(rest)
		return dav, nil
	}
	return nil, fmt.Errorf("Unexpected client type '%s'", conf.Type)
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
