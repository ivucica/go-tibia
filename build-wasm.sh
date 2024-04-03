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
elif [[ -e /usr/local/bin/tinygo ]] ; then
  OUT_TINYGO_WASI_OPT="${GOPATH}/src/badc0de.net/pkg/go-tibia/datafiles/html/main-wasi.wasm"
  OUT_TINYGO_WASI="${GOPATH}/src/badc0de.net/pkg/go-tibia/datafiles/html/main-wasi-default.wasm"
fi

# To run wasi, we could use wasmtime:
# https://github.com/bytecodealliance/wasmtime/blob/9be5dd7c8876a2a3ea12c0cbe2999715e2cc4a22/docs/WASI-tutorial.md

PKG="badc0de.net/pkg/go-tibia/cmd/gotwebfe"

cp "$(go env GOROOT)/misc/wasm/wasm_exec.js" "${GOPATH}/src/badc0de.net/pkg/go-tibia/datafiles/html/wasm_exec.js"

echo "building ${OUT_DEFAULT}"
GOOS=js GOARCH=wasm go build -v -o "${OUT_DEFAULT}" "${PKG}"
echo "built ${OUT_DEFAULT}"

if [[ ! -z "${OUT_WASI}" ]] ; then
  echo "building ${OUT_WASI}"
  GOOS=wasip1 GOARCH=wasm go build -v -o "${OUT_WASI}" "${PKG}"
  echo "built ${OUT_WASI}"
elif [[ ! -z "${OUT_TINYGO_WASI}" ]] ; then
  echo "tinygoing ${OUT_TINYGO_WASI}"
  tinygo build -o "${OUT_TINYGO_WASI}" -target wasi -x "${PKG}"  # remove -x for less output
  echo "tinygoed ${OUT_TINYGO_WASI}"
else
  echo "no GOOS=wasip1 support in $( . <(go env) ; echo ${GOVERSION} ) -- minimum 1.21 per https://go.dev/blog/wasi; skipping WASI build"
fi

function optmoves() {
  mv -v "${OUT_DEFAULT}" "${OUT_OPT}"
  echo "moved, now checking the rest"
  if [[ -e "${OUT_WASI}" ]] ; then
    mv -v "${OUT_WASI}" "${OUT_WASI_OPT}"
  else
    echo "missing wasi default ${OUT_WASI}"
  fi
  if [[ -e "${OUT_TINYGO_WASI}" ]] ; then
    mv -v "${OUT_TINYGO_WASI}" "${OUT_TINYGO_WASI_OPT}"
  else
    echo "missing tinygo wasi default ${OUT_TINYGO_WASI}"
  fi
}

if [[ -z $OPT ]] ; then
  echo "opt switched off"
  optmoves
elif [[ -e /usr/bin/wasm-opt ]] ; then
  echo "wasm-opt found; building ${OUT_OPT}"
  wasm-opt "${OUT_DEFAULT}" --enable-bulk-memory -Oz -o "${OUT_OPT}"
  #rm "${OUT_DEFAULT}"

  if [[ -e "${OUT_WASI}" ]] ; then
    echo "wasm-opt found; building ${OUT_WASI_OPT}"
    wasm-opt "${OUT_WASI}" --enable-bulk-memory -Oz -o "${OUT_WASI_OPT}"
    #rm "${OUT_WASI}"
  fi
  if [[ -e "${OUT_TINYGO_WASI}" ]] ; then
    echo "wasm-opt found; building ${OUT_TINYGO_WASI_OPT}"
    wasm-opt "${OUT_TINYGO_WASI}" --enable-bulk-memory -Oz -o "${OUT_TINYGO_WASI_OPT}"
    #rm "${OUT_TINYGO_WASI}"
  fi
else
  echo "no wasm-opt; moving ${OUT_DEFAULT} -> ${OUT_OPT}"
  optmoves
fi
