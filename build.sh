#!/bin/bash

PWD="$(pwd)"
ROOT="$( cd "$(dirname "$0")" >/dev/null 2>&1 ; pwd -P )"

cd "$ROOT/cmd/davsync"

export CGO_ENABLED=0
go build -a -o "$ROOT/bin/davsync" -tags netgo

cd "$PWD"