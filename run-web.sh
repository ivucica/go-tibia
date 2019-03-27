set -e
export GOPATH=${HOME}/projects/gotserv
go get -v badc0de.net/pkg/go-tibia/cmd/gotweb
${GOPATH}/bin/gotweb --logtostderr --listen_address :9444 "$@"

