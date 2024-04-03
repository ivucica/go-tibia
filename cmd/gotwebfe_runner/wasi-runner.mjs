#!/usr/bin/node --es-module-specifier-resolution=node

//'use strict';

// Looks like I am missing a lot on how NodeJS, ES modules, imports and requires are supposed to work.
// npm.runkit.com, for example, knows nothing of 'import', local nodejs v18.13.0 is complaining in various ways.
//
// Since the actual goal is to run WASI from Go, let's just get this over with in any way possible, and investigate a proper way to run WASI software at a later time.

//console.log('import wasi');
//import { WASI } from "wasi-js";
////const wasiJs = require("wasi-js")
////const WASI = wasiJs.default;

import pkg from 'wasi-js';
const WASI = pkg.default;
//console.log('pkg default:')
//console.log(Object.keys(pkg.default))
//console.log(typeof pkg.default);
//const WASI  = require("wasi-js").default;

//console.log('import fs');

//import fs from "fs";
//const fs = require("fs");
import fs from "node:fs/promises";
import fss from "node:fs";

//import nodeBindings from "wasi-js/dist/bindings/node.js"; // sample code says .../bindings/node, why do we need node.js?
import nodeBindingsI from "wasi-js/dist/bindings/node"; // sample code says .../bindings/node, why do we need node.js?
//const nodeBindings = require("wasi-js/dist/bindings/node.js").default; // sample code says .../bindings/node, why do we need node.js?
//const nodeBindings = require("wasi-js/dist/bindings/node").default;
const nodeBindings = nodeBindingsI.default;

//const path = require("path");
//import path from "path";

//console.log(WASI);


const wasi = new WASI({
  args: [],
  env: {},
  bindings: {...nodeBindings, fs: fss}, //fs},
});
/*
const source = await readFile("datafiles/html/main-wasi.wasm");
const typedArray = new Uint8Array(source);
const result = await WebAssembly.instantiate(typedArray, wasmOpts);
wasi.start(result.instance);
*/

const readFile = fs.readFile; // was the non-functional import above supposed to offload these into the global namespace or something?
const readFileSync = fss.readFileSync; // :)

//const join = path.join; // same -- why do i need to do this when sample code for wasi-js did not need it?
  
//console.log(Object.keys(wasi.wasiImport));


await (async () => {
  const f = await readFile('../../datafiles/html/main-wasi.wasm');
  //const f = await readFile('../../wasmerio-example-module.wasm');
  //console.log(Object.keys(f));
  const wasm = //new Uint8Array(
   await WebAssembly.compile(
    //await readFile(join(__dirname, '../../datafiles/html/main-wasi.wasm')),
    f,
    //await fs.readFileSync(join(__dirname, '../../datafiles/html/main-wasi.wasm')),
    //fs.readFileSync('../../datafiles/html/main-wasi.wasm'),
  );
  //console.log(Object.keys(wasm));
  //console.log(Object.keys(wasi.wasiImport));
  
  const instance = await WebAssembly.instantiate(wasm, { wasi_snapshot_preview1: wasi.wasiImport }); // wasi.getImportObject());

  wasi.start(instance);
})();
