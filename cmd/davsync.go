package main

import (
	"encoding/json"
	"flag"
	"io/ioutil"
	"log"

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

func createRemoteClient(args Args) model.Client {
	return createRemoteDavClient(args)
}

func createRemoteDavClient(args Args) *webdav.Client {
	adapter := webdav.NewAdapter()
	adapter.BaseURI = args.secrets.BaseURI
	adapter.BasePath = args.remotePath
	adapter.AuthToken = args.secrets.Token
	adapter.AuthTokenType = args.secrets.TokenType
	adapter.AuthUser = args.secrets.User
	adapter.AuthPass = args.secrets.Pass
	return webdav.NewClient(adapter)
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
	/*
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

		log.Println("Trrying upload file..")

		re, yaerr := regexp.Compile("(^|/+)[^/]+$")
		if yaerr != nil {
			log.Fatal(yaerr)
		}
		log.Println("Replaced", re.ReplaceAllString("/foo/bar/baz/test-file-big.bin", ""))
		log.Println("Replaced", re.ReplaceAllString("/test-file-big.bin", ""))
		log.Println("Replaced", re.ReplaceAllString("test-file-big.bin", ""))
		log.Println("Replaced", re.ReplaceAllString("/1/test-file-big.bin", ""))
		log.Println("Replaced", re.ReplaceAllString("1/test-file-big.bin", ""))

		//f, yaerr = os.Open("/home/iodev/projects/local/davsync/.dav-test-input/code-stable-1585036655.tar.gz")
		f, yaerr = os.Open("/tmp/davsync.out")
		if yaerr != nil {
			log.Fatal(yaerr)
		}

		finfo, yaerr := f.Stat()
		if yaerr != nil {
			log.Fatal(yaerr)
		}
		readProgress := model.NewReadProgress(f, finfo.Size())

		log.Println("  mkdir..")
		yaerr = yaclient.MakeDirFor("/foo/bar/baz/test-file-big.bin")
		if yaerr != nil {
			log.Fatal(yaerr)
		}
		log.Println("  uploading..")
		yaerr = yaclient.WriteFile("/foo/bar/baz/test-file-big.bin", readProgress)
		if yaerr != nil {
			log.Fatal(yaerr)
		}
		f.Close()

		log.Println("Done.")

		return
	*/
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
