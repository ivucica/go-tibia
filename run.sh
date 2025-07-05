#!/bin/bash
set -e
export GOPATH="${GOPATH:-${HOME}/projects/go-tibia}"
go get -v badc0de.net/pkg/go-tibia/cmd/gotserv
go install -v badc0de.net/pkg/go-tibia/cmd/gotserv
${GOPATH}/bin/gotserv --logtostderr \
    "${PORT_LOGIN:+--login_listen_address "${PORT_LOGIN}"}" \
    "${PORT_GAME:+--game_listen_address "${PORT_GAME}"}" \
    "${PORT:+--debug_web_server_listen_address "${PORT}"} \
    "$@"

