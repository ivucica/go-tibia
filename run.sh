set -e
export GOPATH=${HOME}/Projects/go-tibia
go get -v badc0de.net/pkg/go-tibia/cmd/gotserv
${GOPATH}/bin/gotserv --logtostderr "$@"

