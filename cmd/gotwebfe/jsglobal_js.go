//go:build js && wasm && !tinygo.wasm
// +build js,wasm,!tinygo.wasm

package main

import (
	"fmt"
	"log"
	"runtime/debug"
	"syscall/js"
)

// alias for js.Value in browser and non-WASI WASM is js.Value
type jsValue = js.Value

var shareUIs int

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

	// Register message handling for messages from a service worker:
	// https://stackoverflow.com/a/42964575/39974
	//
	// It is correct to register on navigator.serviceWorker.
	//
	// Currently mainly used to receive share target messages when the service
	// worker receives a share request.
	js.Global().Get("navigator").Get("serviceWorker").Call(
		"addEventListener",
		"message",
		js.FuncOf(handleSharedMessage))
	js.Global().Set("handleSharedMessage", js.FuncOf(handleSharedMessage))
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

// handleSharedMessage handles "message" events from the service worker when the
// the web app is a share target and the user shares something to it.
func handleSharedMessage(this jsValue, args []jsValue) interface{} {
	event := args[0]
	data := event.Get("data")
	dataType := data.Get("type").String()

	if dataType != "share" {
		return nil
	}

	shareUIs++
	uiID := shareUIs

	title := data.Get("title").String()
	text := data.Get("text").String()
	var fileImg string
	for _, field := range []string{"jpegs", "pngs", "graphs", "images"} {
		if data.Get(field).Truthy() && data.Get(field).String() != "" {
			fileImg = data.Get(field).String()
			// TODO: What if we get more than one file in the field? what if we get more than one field?
		}
	}
	var fileOther string
	for _, field := range []string{"plaintextfiles", "records", "textfiles", "otherfiles"} {
		if data.Get(field).Truthy() && data.Get(field).String() != "" {
			fileOther = data.Get(field).String()
		}
	}
	sharedURL := data.Get("url").String()

	// TODO: abstract away creating windows in general. since this is the only
	// one we dynamically create, it is ok for now.
	document := jsGlobal().Get("document")

	window := document.Call("createElement", "div")
	window.Set("className", "window")
	window.Get("style").Set("width", "300px")
	window.Get("style").Set("margin", "10px auto")
	window.Set("id", fmt.Sprintf("share-window-%d", uiID))

	titlebar := document.Call("createElement", "div")
	titlebar.Set("className", "titlebar")
	titlebar.Set("innerHTML", "Share Target")
	window.Call("appendChild", titlebar)

	content := document.Call("createElement", "div")
	content.Set("className", "content")
	window.Call("appendChild", content)

	if title != "" {
		pTitle := document.Call("createElement", "p")
		pTitle.Set("innerHTML", "Title: "+title)
		content.Call("appendChild", pTitle)
	}
	if text != "" {
		pText := document.Call("createElement", "p")
		pText.Set("innerHTML", "Text: "+text)
		content.Call("appendChild", pText)
	}
	if sharedURL != "" {
		pURL := document.Call("createElement", "p")
		pURL.Set("innerHTML", "URL: "+text)
		content.Call("appendChild", pURL)
	}

	if fileImg != "" {
		img := document.Call("createElement", "img")
		img.Set("src", fileImg)
		img.Get("style").Set("maxWidth", "100%")
		content.Call("appendChild", img)
	}
	if fileOther != "" {
		a := document.Call("createElement", "a")
		a.Set("href", fileOther)
		a.Set("innerHTML", "Other file")
		content.Call("appendChild", a)
	}

	bottombar := document.Call("createElement", "div")
	bottombar.Set("className", "bottombar")
	window.Call("appendChild", bottombar)

	buttonArea := document.Call("createElement", "div")
	buttonArea.Get("style").Set("textAlign", "center")
	content.Call("appendChild", buttonArea)

	shareButton := document.Call("createElement", "button")
	shareButton.Set("innerHTML", "<div>Share</div>")
	buttonArea.Call("appendChild", shareButton)

	closeButton := document.Call("createElement", "button")
	closeButton.Set("innerHTML", "<div>Close</div>")
	closeButton.Call("addEventListener", "click", js.FuncOf(func(this jsValue, args []jsValue) interface{} {
		window.Call("remove")
		return nil
	}))
	buttonArea.Call("appendChild", closeButton)

	container := document.Call("getElementById", "share-target-container")
	container.Call("appendChild", window)

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
