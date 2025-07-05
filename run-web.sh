#!/bin/bash
set -e

if [[ ! -z ${IDX_ENV_CONFIG_FILE_PATH} ]] && [[ ! -z "$(command -v jq)" ]] ; then
	WORKSPACE="$(jq -r .config.workspaceFolder "${IDX_ENV_CONFIG_FILE_PATH}")"
	if [[ -e "${WORKSPACE}/run-web.sh" ]] ; then
		mkdir -p "${HOME}/projects/go-tibia/src/badc0de.net/pkg"
		ln -s "${WORKSPACE}" "${HOME}"/projects/go-tibia/src/badc0de.net/pkg/go-tibia
	fi
	if [[ ! -e "datafiles/Tibia.spr" ]] ; then
		bazelisk build @tibia854//:Tibia.spr
	fi
fi

export GOPATH="${GOPATH:-${HOME}/projects/go-tibia}"


VAPID_ARGS=
if [[ -e "${GOPATH}"/src/badc0de.net/pkg/go-tibia/vapid.inc.sh ]] ; then
	 . "${GOPATH}"/src/badc0de.net/pkg/go-tibia/vapid.inc.sh
	VAPID_ARGS="--vapid_private=${GOTIBIA_VAPID_PRIVATE} --vapid_public=${GOTIBIA_VAPID_PUBLIC}"
fi

if [[ ! -z ${REBUILD} ]] ; then
go get -v badc0de.net/pkg/go-tibia/cmd/gotweb
go install -v badc0de.net/pkg/go-tibia/cmd/gotweb
if [[ ${WASM_BUILD} != off ]] ; then
  ${GOPATH}/src/badc0de.net/pkg/go-tibia/build-wasm.sh
fi
fi

${GOPATH}/bin/gotweb --logtostderr --listen_address :${PORT:-9444} ${VAPID_ARGS} --flag_file=${GOPATH}/src/badc0de.net/pkg/go-tibia/gotweb.flagfile "$@"
