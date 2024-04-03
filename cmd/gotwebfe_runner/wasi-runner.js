#!/usr/bin/env nodejs
'use strict';

// Looks like I am missing a lot on how NodeJS, ES modules, imports and requires are supposed to work.
// npm.runkit.com, for example, knows nothing of 'import', local nodejs v18.13.0 is complaining in various ways.
//
// Since the actual goal is to run WASI from Go, let's just get this over with in any way possible, and investigate a proper way to run WASI software at a later time.

//import { WASI } from "wasi-js";
const wasiJs = require("wasi-js")
const WASI = wasiJs.default;

//import fs from "fs";
const fs = require("fs");

//import nodeBindings from "wasi-js/dist/bindings/node.js"; // sample code says .../bindings/node, why do we need node.js?
const nodeBindings = require("wasi-js/dist/bindings/node.js").default; // sample code says .../bindings/node, why do we need node.js?

const path = require("path");

console.log(WASI);
const wasi = new WASI({
  args: [],
  env: {},
  bindings: {...nodeBindings, fs},
});
/*
const source = await readFile("datafiles/html/main-wasi.wasm");
const typedArray = new Uint8Array(source);
const result = await WebAssembly.instantiate(typedArray, wasmOpts);
wasi.start(result.instance);
*/

const readFile = fs.readFile; // was the non-functional import above supposed to offload these into the global namespace or something?

const join = path.join; // same -- why do i need to do this when sample code for wasi-js did not need it?

  console.log(wasi);


(async () => {
  const wasm = await WebAssembly.compile(
    //await readFile(join(__dirname, '../../datafiles/html/main-wasi.wasm')),
    await fs.readFileSync(join(__dirname, '../../datafiles/html/main-wasi.wasm')),
  );
  const instance = await WebAssembly.instantiate(wasm, wasi.getImportObject());

  wasi.start(instance);
})();
