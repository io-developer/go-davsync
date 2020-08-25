package main

import (
	"encoding/json"
	"flag"
	"io/ioutil"
	"log"

	"github.com/io-developer/go-davsync/pkg/client"
	"github.com/io-developer/go-davsync/pkg/client/fs"
	"github.com/io-developer/go-davsync/pkg/client/webdav"
	"github.com/io-developer/go-davsync/pkg/client/yadiskrest"
	"github.com/io-developer/go-davsync/pkg/model"
)

type Args struct {
	localPath   string
	remotePath  string
	secretsFile string
	secrets     Secrets
}

type Secrets struct {
	BaseURI   string
	Token     string
	TokenType string
	User      string
	Pass      string
}

func parseArgs() Args {
	localPath := flag.String("local", "./", "Local directory path. Example: /tmp/test")
	remotePath := flag.String("remote", "/", "Webdav directory path. Example: /test")
	secretsFile := flag.String("secrets", ".davsync", "JSON config for base URI and auth secrets")
	flag.Parse()

	args := Args{
		localPath:   *localPath,
		remotePath:  *remotePath,
		secretsFile: *secretsFile,
	}

	if *secretsFile != "" {
		bytes, err := ioutil.ReadFile(*secretsFile)
		log.Println("bytes", string(bytes))
		if err != nil {
			log.Fatal(err)
		}
		secrets := Secrets{}
		err = json.Unmarshal(bytes, &secrets)
		if err != nil {
			log.Fatal(err)
		}
		args.secrets = secrets

		log.Println("secrets", secrets)
	}

	return args
}

func createSrcClient(args Args) *fs.Client {
	return createFsClient(args)
}

func createDstClient(args Args) client.Client {
	return createRemoteDavClient(args)
}

func createFsClient(args Args) *fs.Client {
	return fs.NewClient(args.localPath)
}

func createRemoteDavClient(args Args) *webdav.Client {
	return webdav.NewClient(webdav.Options{
		BaseDir:       args.remotePath,
		DavUri:        args.secrets.BaseURI,
		AuthToken:     args.secrets.Token,
		AuthTokenType: args.secrets.TokenType,
		AuthUser:      args.secrets.User,
		AuthPass:      args.secrets.Pass,
	})
}

func createRemoteYaClient(args Args) *yadiskrest.Client {
	opt := yadiskrest.ClientOpt{
		ApiUri:    "https://cloud-api.yandex.net/v1/disk",
		AuthToken: args.secrets.Token,
	}
	client := yadiskrest.NewClient(opt)
	client.BaseDir = args.remotePath
	return client
}

func main() {
	args := parseArgs()
	log.Printf("CLI ARGS:\n%#v\n\n", args)

	src := createSrcClient(args)
	dst := createDstClient(args)
	sync := model.NewSync1Way(src, dst, model.Sync1WayOpt{
		IndirectUpload: true,
		IgnoreExisting: true,
		AllowDelete:    false,
		WriteThreads:   4,
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
