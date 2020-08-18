package main

import (
	"encoding/json"
	"flag"
	"io/ioutil"
	"log"

	"github.com/io-developer/davsync/client/fs"
	"github.com/io-developer/davsync/client/webdav"
	"github.com/io-developer/davsync/model"
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

func createLocalClient(args Args) *fs.Client {
	return fs.NewClient(args.localPath)
}

func createRemoteClient(args Args) *webdav.Client {
	adapter := webdav.NewAdapter()
	adapter.BaseURI = args.secrets.BaseURI
	adapter.BasePath = args.remotePath
	adapter.AuthToken = args.secrets.Token
	adapter.AuthTokenType = args.secrets.TokenType
	adapter.AuthUser = args.secrets.User
	adapter.AuthPass = args.secrets.Pass
	return webdav.NewClient(adapter)
}

func main() {
	args := parseArgs()
	log.Printf("CLI ARGS:\n%#v\n\n", args)

	local := createLocalClient(args)
	remote := createRemoteClient(args)

	sync := model.NewSync1Way(local, remote, model.Sync1WayOpt{
		IgnoreExisting: true,
		AllowDelete:    false,
	})

	err := sync.Sync()
	if err != nil {
		log.Panicln("sync Sync()", err)
	}

	log.Println("\n\nDone.")
}
