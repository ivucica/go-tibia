set -e
go get -v badc0de.net/pkg/go-tibia/cmd/gotserv
${GOPATH}/bin/gotserv --logtostderr "$@"

