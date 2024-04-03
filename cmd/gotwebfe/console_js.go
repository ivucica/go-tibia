//go:build js && wasm && !tinygo.wasm
// +build js,wasm,!tinygo.wasm

package main

import (
	"syscall/js"
)

func (*console) Write(p []byte) (n int, err error) {
	js.Global().Get("window").Get("console").Call("log", js.ValueOf(string(p)))
	return len(p), nil
}
