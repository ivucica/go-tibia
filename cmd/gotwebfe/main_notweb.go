//go:build !js && !wasm && !tinygo.wasm
// +build !js,!wasm,!tinygo.wasm

package main

func main() {
	panic("this only runs on web (build with wasm, please)")
}
