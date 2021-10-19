#!/bin/bash
set -e
export GOPATH="${GOPATH:-${HOME}/projects/go-tibia}"
go get -v badc0de.net/pkg/go-tibia/cmd/gotweb
${GOPATH}/src/badc0de.net/pkg/go-tibia/build-wasm.sh
${GOPATH}/bin/gotweb --logtostderr --listen_address :9444 "$@"

