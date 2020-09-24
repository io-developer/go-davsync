#!/bin/bash

docker run --rm -v $(pwd):/app golang:1.15.2 /app/build.sh