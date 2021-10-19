#!/bin/bash
set -e
export GOPATH="${GOPATH:-${HOME}/projects/go-tibia}"
go get -v badc0de.net/pkg/go-tibia/cmd/gotserv
${GOPATH}/bin/gotserv --logtostderr "$@"

