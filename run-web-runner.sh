#!/bin/bash
set -e
export GOPATH="${GOPATH:-${HOME}/projects/go-tibia}"

go get -v badc0de.net/pkg/go-tibia/cmd/gotwebfe_runner
go install -v badc0de.net/pkg/go-tibia/cmd/gotwebfe_runner
if [[ ${WASM_BUILD} != off ]] ; then
  ${GOPATH}/src/badc0de.net/pkg/go-tibia/build-wasm.sh
fi
(
# Remove the cd and the () once the path to main.wasm is not hardcoded.
cd ${GOPATH}/src/badc0de.net/pkg/go-tibia
${GOPATH}/bin/gotwebfe_runner "$@"
)
