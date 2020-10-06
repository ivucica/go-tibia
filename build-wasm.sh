#!/bin/bash

if [[ -z $GOPATH ]] ; then
    echo 'GOPATH must be set'
    exit 1
fi

cp "$(go env GOROOT)/misc/wasm/wasm_exec.js" "${GOPATH}/src/badc0de.net/pkg/go-tibia/datafiles/html/wasm_exec.js"
GOOS=js GOARCH=wasm go build -v -o "${GOPATH}/src/badc0de.net/pkg/go-tibia/datafiles/html/main.wasm" badc0de.net/pkg/go-tibia/cmd/gotwebfe
