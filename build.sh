#!/bin/bash

export CGO_ENABLED=0

go build -a -o "$(pwd)/bin/davsync" -tags netgo "./cmd/davsync.go"
