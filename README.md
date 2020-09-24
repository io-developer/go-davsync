# go-davsync
Rsync-style WebDAV synchronizer

[![Docker Automated build](https://img.shields.io/docker/automated/iodeveloper/go-davsync.svg)](https://registry.hub.docker.com/r/iodeveloper/go-davsync)

[![Docker Hub repository](http://dockeri.co/image/iodeveloper/go-davsync)](https://registry.hub.docker.com/r/iodeveloper/go-davsync)

`iodeveloper/go-davsync:latest`

## Usage
* `-i /some/input/dir` - path of source directory. Default is local `./`
* `-iconf /input/config.json` - path to secrets and options. Default none - means local filesystem source
* `-o /some/output/dir` - path of target directory. Default `/`
* `-oconf /output/config.json` path to secrets and options. Default `.davsync` in workdir

## Example
* Suppose you have to sync out some local files into remote WebDAV folder
* Put local files into `./files/`. Structure in example: 
```
> tree ./files

files
├── 123
│   ├── 456
│   │   ├── apcupsd-3.14.14.tar.gz
│   │   └── hp-laserjet-1018.zip
│   └── unison-2.51.2.tar.gz
├── bar
│   └── baz
│       └── mipsel.bin
├── davsync-old-backup-2.log
├── davsync-old-backup-5.log
├── davsync-old-backup-6.log
├── davsync-old-backup.log
├── F50SLAS209.zip
├── foo
│   └── F50SLAS209.zip
├── info.log
├── mipsel.bin
└── vscode.tar.gz

```
* Put remote config to `./davconf.json`. For example:
```json
{
    "Type": "Webdav",
    "WebdavOptions": {
        "DavUri": "https://webdav.yandex.ru/",
        "AuthUser": "YOUR_USER",
        "AuthPass": "YOUR_PASSWORD"
    },
}
```
* Specify remote destination directory with `-o /Uploads/01/`
* Set desired max parallel threads `-threads 4`
* Run
```bash
docker run --rm                             \
    -v $(pwd)/files:/input                  \
    -v $(pwd)/davconf.json:/davconf.json    \
    iodeveloper/go-davsync:latest           \
        -i /input                           \
        -o "/Uploads/01"                    \
        -oconf /davconf.json                \
        -threads 4
```
* Output should be like:
```
Sync: Making dirs...
Sync:   make dir /
Sync:   make dir /123/
Sync:   make dir /123/456/
Sync:   make dir /bar/
Sync:   make dir /bar/baz/
Sync:   make dir /foo/
Sync: UPL   0.0%   0/0:  Handling...
Sync: UPL   0.0%   0/19:  Complete                          /                               
Sync: UPL   5.3%   1/19:  Complete                          /123/                           
Sync: UPL  10.5%   2/19:  Complete                          /123/456/                       
Sync: UPL  15.8%   3/19:  Complete                          /F50SLAS209.zip                 
Sync: UPL  21.1%   4/19:  Complete                          /bar/                           
Sync: UPL  26.3%   5/19:  Complete                          /bar/baz/                       
Sync: UPL  31.6%   6/19:  Complete                          /123/unison-2.51.2.tar.gz       
Sync: UPL  36.8%   7/19:  Complete                          /123/456/hp-laserjet-1018.zip   
Sync: UPL  42.1%   8/19:  Complete                          /123/456/apcupsd-3.14.14.tar.gz 
Sync: UPL  47.4%   9/19:  Complete                          /bar/baz/mipsel.bin             
Sync: UPL  52.6%  10/19:  Complete                          /davsync-old-backup-6.log       
Sync: UPL  57.9%  11/19:  Complete                          /foo/                           
Sync: UPL  63.2%  12/19:  Complete                          /davsync-old-backup-2.log       
Sync: UPL  68.4%  13/19:  Complete                          /davsync-old-backup.log         
Sync: UPL  73.7%  14/19:  Complete                          /foo/F50SLAS209.zip             
Sync: UPL  78.9%  15/19:  Complete                          /info.log                       
Sync: UPL  84.2%  16/19:  Complete                          /mipsel.bin                     
Sync: UPL  89.5%  17/19:  Complete                          /davsync-old-backup-5.log       
Sync: UPL  94.7%  18/19:  Complete                          /vscode.tar.gz                  
Sync: UPL 100.0%  19/19:  Complete
Sync: DEL   0.0%   0/0:  Handling...
Sync: DEL   0.0%   0/0:  Nothing to do
Sync: Listening thread logs finished


Done.
```

## Other examples

### Local to DAV
```bash
bin/davsync -i /local/input -o /dav/output -oconf /tmp/oconf.json
```

User/pass auth
`/tmp/oconf.json`
```json
{
    "Type": "Webdav",
    "WebdavOptions": {
        "DavUri": "https://webdav.yandex.ru/",
        "AuthUser": "YOUR_USER",
        "AuthPass": "YOUR_PASSWORD"
    },
}
```

Or token auth
`/tmp/oconf.json`
```json
{
    "Type": "Webdav",
    "WebdavOptions": {
        "DavUri": "https://webdav.yandex.ru/",
        "AuthToken": "YOUR_TOKEN",
        "AuthTokenType": "OAuth"
    },
}
```

### Local to YandexDisk
Pure REST is slower than composition of DAV for files + REST for file-tree & hashes

```bash
bin/davsync -i /local/input -o /yadisk/output -oconf /tmp/oconf.json
```
`/tmp/oconf.json`
```json
{
    "Type": "Yadisk",
    "WebdavOptions": {
        "DavUri": "https://webdav.yandex.ru/",
        "AuthToken": "YOUR_TOKEN",
        "AuthTokenType": "OAuth"
    },
    "YadiskRestOptions": {
        "ApiUri": "https://cloud-api.yandex.net/v1/disk",
        "AuthToken": "YOUR_TOKEN",
        "AuthTokenType": "OAuth"
    }
}
```

### DAV to local
Be sure input and ouput directories are exist
```bash
bin/davsync -i /dav/input -iconf /tmp/oconf.json -o /local/output -oconf /tmp/oconf.json
```
`/tmp/iconf.json`
```json
{
    "Type": "Webdav",
    "WebdavOptions": {
        "DavUri": "https://webdav.yandex.ru/",
        "AuthToken": "YOUR_TOKEN",
        "AuthTokenType": "OAuth"
    },
}
```
`/tmp/oconf.json`
```json
{
    "Type": "Local"
}
```
