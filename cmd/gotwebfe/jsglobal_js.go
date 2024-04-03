//go:build js && wasm && !tinygo.wasm
// +build js,wasm,!tinygo.wasm

package main

import (
	"log"
	"runtime/debug"
	"syscall/js"
)

// alias for js.Value in browser and non-WASI WASM is js.Value
type jsValue = js.Value

// Prevent go program from exiting.
func persistIfBrowser() {
	select {}
}

func jsGlobalInjectAPI() {
	js.Global().Set("goVar", "I am a variable set from Go")
	js.Global().Set("sayHello", js.FuncOf(sayHello))

	js.Global().Set("showSpr", js.FuncOf(showSpr))
	js.Global().Set("showMap", js.FuncOf(showMap))
	js.Global().Set("loaderPromise", js.FuncOf(loaderPromise))
	js.Global().Set("addX", js.FuncOf(addX))
	js.Global().Set("addY", js.FuncOf(addY))
	js.Global().Set("subX", js.FuncOf(subX))
	js.Global().Set("subY", js.FuncOf(subY))
}

func jsGlobal() js.Value {
	return js.Global()
}

func addX(this js.Value, arg []js.Value) interface{} {
	tx++
	return nil
}

func addY(this js.Value, arg []js.Value) interface{} {
	ty++
	return nil
}

func subX(this js.Value, arg []js.Value) interface{} {
	tx--
	return nil
}

func subY(this js.Value, arg []js.Value) interface{} {
	ty--
	return nil
}

func loaderPromise(this jsValue, arg []jsValue) interface{} {
	// https://withblue.ink/2020/10/03/go-webassembly-http-requests-and-promises.html
	log.Println("constructing a loader promise")
	defer log.Println("constructed a loader promise")
	promiseConstructor := jsGlobal().Get("Promise")
	return promiseConstructor.New(js.FuncOf(loaderPromiseImp))
}

func debugFreeOSMemory() {
	debug.FreeOSMemory()
}

func showDOM(showable jsValue, parentElementID string, replace bool) {
	document := jsGlobal().Get("document")

	parent := document.Call("getElementById", parentElementID)
	if parent.IsNull() || parent.IsUndefined() {
		log.Printf("[W] showImg: appending to body, because %q cannot be found", parentElementID)
		parent = document.Get("body")
	} else if replace {
		safety := 50
		for {
			lastChild := parent.Get("lastChild")
			if !lastChild.Truthy() {
				break
			}
			parent.Call("removeChild", parent.Get("lastChild"))
			safety--
			if safety <= 0 {
				log.Printf("[W] showImg: more than 50 children or a bug; aborting replace")
				break
			}
		}
	}
	parent.Call("appendChild", showable)
}
