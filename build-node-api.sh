#!/bin/bash

# Check if go is installed
if ! [ -x "$(command -v go)" ]; then
  echo 'Error: go is not installed.' >&2
  exit 1
fi

mkdir build

go build -o build/node-api cmd/go-api/main.go