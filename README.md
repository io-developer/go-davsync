# go-davsync
Rsync-style WebDAV synchronizer

## Usage
* `-i /some/input/dir` - path of source directory. Default is local `./`
* `-iconf /input/config.json` - path to secrets and options. Default none - means local filesystem source
* `-o /some/output/dir` - path of target directory. Default `/`
* `-oconf /output/config.json` path to secrets and options. Default `.davsync` in workdir

### Example. Local to DAV
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

### Example. Local to YandexDisk
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

### Example. DAV to local
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
