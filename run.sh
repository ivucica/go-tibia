set -e
go get -v badc0de.net/pkg/gotserv/cmd/gotserv
${GOPATH}/bin/gotserv --logtostderr

