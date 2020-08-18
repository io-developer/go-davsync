package main

import (
	"encoding/json"
	"flag"
	"io"
	"io/ioutil"
	"log"
	"os"

	"github.com/io-developer/davsync/client/fs"
	"github.com/io-developer/davsync/client/webdav"
	"github.com/io-developer/davsync/client/yadiskrest"
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

	yaopt := yadiskrest.ClientOpt{
		ApiUri:    "https://cloud-api.yandex.net/v1/disk",
		AuthToken: args.secrets.Token,
	}
	yaclient := yadiskrest.NewClient(yaopt)
	yaclient.BaseDir = "/Загрузки"

	yapaths, yanodes, yaerr := yaclient.ReadTree()
	_, yaerr = yaclient.GetResources()
	if yaerr != nil {
		log.Fatal(yaerr)
	}

	for _, yanode := range yanodes {
		log.Println("\nNODE")
		log.Println("  Path", yanode.Path)
		log.Println("  AbsPath", yanode.AbsPath)
		log.Println("  Name", yanode.Name)
		log.Println("  IsDir", yanode.IsDir)
		log.Println("  Size", yanode.Size)
		log.Printf("  UserData %#v\n", yanode.UserData)
	}

	for _, yapath := range yapaths {
		log.Println(yapath)
	}

	log.Println("Trrying get file..")
	yareader, yaerr := yaclient.ReadFile("/test-note3/F50SLAS209.zip")
	if yaerr != nil {
		log.Fatal(yaerr)
	}
	f, yaerr := os.OpenFile("/tmp/davsync.out", os.O_CREATE|os.O_WRONLY, 0644)
	if yaerr != nil {
		log.Fatal(yaerr)
	}
	_, yaerr = io.Copy(f, yareader)
	if yaerr != nil {
		log.Fatal(yaerr)
	}
	return
	log.Println("Trrying upload file..")
	f, yaerr = os.Open("/home/iodev/projects/local/davsync/.dav-test-input/code-stable-1585036655.tar.gz")
	if yaerr != nil {
		log.Fatal(yaerr)
	}

	finfo, yaerr := f.Stat()
	if yaerr != nil {
		log.Fatal(yaerr)
	}
	readProgress := model.NewReadProgress(f, finfo.Size())

	yaerr = yaclient.WriteFile("/test-file-big.bin", readProgress)
	if yaerr != nil {
		log.Fatal(yaerr)
	}
	f.Close()

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
