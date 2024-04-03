#!/usr/bin/node --es-module-specifier-resolution=node
// For NodeJS v18.13.0

import wasiJs from 'wasi-js';
const WASI = wasiJs.default;

import fs from "node:fs/promises";
import fss from "node:fs";

import nodeBindingsI from "wasi-js/dist/bindings/node";
const nodeBindings = nodeBindingsI.default;

const wasi = new WASI({
  args: [],
  env: {},
  bindings: {...nodeBindings, fs: fss},
});

/*
const source = await readFile("datafiles/html/main-wasi.wasm");
const typedArray = new Uint8Array(source);
const result = await WebAssembly.instantiate(typedArray, wasmOpts);
wasi.start(result.instance);
*/

const readFile = fs.readFile;
const readFileSync = fss.readFileSync;

await (async () => {
  const wasm = await WebAssembly.compile(
    await readFile('../../datafiles/html/main-wasi.wasm'),
  );
  
  const instance = await WebAssembly.instantiate(wasm, { wasi_snapshot_preview1: wasi.wasiImport });

  wasi.start(instance);
})();
