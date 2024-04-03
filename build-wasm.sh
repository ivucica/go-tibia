#!/bin/bash

if [[ -z $GOPATH ]] ; then
    echo 'GOPATH must be set'
    exit 1
fi

OUT_OPT="${GOPATH}/src/badc0de.net/pkg/go-tibia/datafiles/html/main.wasm"
OUT_DEFAULT="${GOPATH}/src/badc0de.net/pkg/go-tibia/datafiles/html/main-default.wasm"

if [[ $( . <(go env) ; echo ${GOVERSION} | cut -d'.' -f2) -ge 21 ]] ; then
  OUT_WASI_OPT="${GOPATH}/src/badc0de.net/pkg/go-tibia/datafiles/html/main-wasi.wasm"
  OUT_WASI="${GOPATH}/src/badc0de.net/pkg/go-tibia/datafiles/html/main-wasi-default.wasm"
fi

PKG="badc0de.net/pkg/go-tibia/cmd/gotwebfe"

cp "$(go env GOROOT)/misc/wasm/wasm_exec.js" "${GOPATH}/src/badc0de.net/pkg/go-tibia/datafiles/html/wasm_exec.js"

echo "building ${OUT_DEFAULT}"
GOOS=js GOARCH=wasm go build -v -o "${OUT_DEFAULT}" "${PKG}"
echo "built ${OUT_DEFAULT}"

if [[ ! -z "${OUT_WASI}" ]] ; then
  echo "building ${OUT_WASI}"
  GOOS=wasip1 GOARCH=wasm go build -v -o "${OUT_WASI}" "${PKG}"
  echo "built ${OUT_WASI}"
else
  echo "no GOOS=wasip1 support in $( . <(go env) ; echo ${GOVERSION} ) -- minimum 1.21 per https://go.dev/blog/wasi; skipping WASI build"
fi

if [[ -z $OPT ]] ; then
  mv -v "${OUT_DEFAULT}" "${OUT_OPT}"
elif [[ -e /usr/bin/wasm-opt ]] ; then
  echo "wasm-opt found; building ${OUT_OPT}"
  wasm-opt "${OUT_DEFAULT}" --enable-bulk-memory -Oz -o "${OUT_OPT}"
  echo "wasm-opt found; building ${OUT_WASI_OPT}"
  wasm-opt "${OUT_WASI}" --enable-bulk-memory -Oz -o "${OUT_WASI_OPT}"
  #rm "${OUT_DEFAULT}"
else
  echo "no wasm-opt; moving"
  mv -v "${OUT_DEFAULT}" "${OUT_OPT}"
  if [[ -e "${OUT_WASI}" ]] ; then
    mv -v "${OUT_WASI}" "${OUT_WASI_OPT}"
  fi
fi
