//go:build tinygo.wasm
// +build tinygo.wasm

package main

import (
	"log"
)

// alias for js.Value in WASI is, for now, interface{}
type jsValue = hackJSValue

type hackJSValue interface {
	Get(s string) jsValue
	Set(s string, v interface{})
	String() string
	Call(string, ...interface{}) jsValue
	IsNull() bool
	IsUndefined() bool
	Truthy() bool
	Int() int
	New(...interface{}) jsValue
	Invoke(...interface{}) // ?
}

func persistIfBrowser() {
	log.Println("not browser; not persisting")
}

func jsGlobalInjectAPI() {
	log.Println("not injecting any api into WASI environment (js.Global() / 'window' object might be something writable into, but not supported right now")
}

func jsGlobal() jsValue {
	log.Println("WASI hack: returning nil for js.Global()")
	return nil
}

func loaderPromise(this jsValue, arg []jsValue) interface{} {
	log.Println("no current support for promises when running under WASI")
	return nil
}

func debugFreeOSMemory() {
	log.Println("debug.FreeOSMemory not defined on WASI on tinygo")
}

func showDOM(showable interface{}, parentElementID string, replace bool) {
	log.Println("showDOM not implemented under tinygo WASI")
}
