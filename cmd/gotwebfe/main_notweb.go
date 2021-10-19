// +build !js,!wasm

package main

func main() {
	panic("this only runs on web (build with wasm, please)")
}
