set -e
export GOPATH=${HOME}/projects/gotserv
go get -v badc0de.net/pkg/go-tibia/cmd/gotserv
${GOPATH}/bin/gotserv --logtostderr "$@"

