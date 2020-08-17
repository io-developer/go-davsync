package main

import (
	"encoding/json"
	"flag"
	"io/ioutil"
	"log"

	"github.com/io-developer/davsync/fs"
	"github.com/io-developer/davsync/model"
	"github.com/io-developer/davsync/webdav"
	//	"github.com/studio-b12/gowebdav"
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
	localPaths, localNodes, err := local.ReadTree()
	if err != nil {
		log.Fatalln("ReadTree err", err)
	}
	logTree(localPaths, localNodes)

	remote := createRemoteClient(args)
	remotePaths, remoteNodes, err := remote.ReadTree()
	if err != nil {
		log.Fatalln("ReadTree err", err)
	}
	logTree(remotePaths, remoteNodes)

	bothPaths, addPaths, delPaths := model.NodeComparePaths(localNodes, remoteNodes)
	for _, path := range bothPaths {
		log.Println("BOTH", path)
	}
	for _, path := range addPaths {
		log.Println("ADD", path)
	}
	for _, path := range delPaths {
		log.Println("DEL", path)
	}

	for _, path := range addPaths {
		node := localNodes[path]
		if node.IsDir {
			log.Println("TRY ADD DIR", path)
			err := remote.Mkdir(path, true)
			if err != nil {
				log.Fatal(err)
			}
		}
	}

	log.Println("\n\nDone.")
}

func logTree(paths []string, nodes map[string]model.Node) {
	for _, path := range paths {
		log.Println(path)
	}
	for path, node := range nodes {
		log.Printf("\n%s\n%#v\n\n", path, node)
	}
}
