// +build js,wasm

package main

import (
	"bytes"
	"fmt"
	"image"
	"image/png"
	"io"
	"net/http"
	"syscall/js"

	"badc0de.net/pkg/go-tibia/spr"
	"github.com/vincent-petithory/dataurl"
)

var sprBytes []byte

func main() {

	response, err := http.Get("Tibia.spr")
	if err != nil {
		showError("opening Tibia.spr", err)
		select {}
	}
	if response.StatusCode != http.StatusOK {
		showError("opening Tibia.spr", fmt.Errorf("status code was %d", response.StatusCode))
		select {}
	}

	buf := &bytes.Buffer{}
	if _, err := io.Copy(buf, response.Body); err != nil {
		showError("copying response to seekable buffer", err)
		select {}
	}
	response.Body.Close()

	sprBytes = buf.Bytes()

	img, err := spr.DecodeOne(bytes.NewReader(buf.Bytes()), 500)
	if err != nil {
		showError("decoding a spr", err)
		select {}
	}

	showImg(img)

	//2. Exposing go functions/values in javascript variables.
	js.Global().Set("goVar", "I am a variable set from Go")
	js.Global().Set("sayHello", js.FuncOf(sayHello))

	js.Global().Set("showSpr", js.FuncOf(showSpr))

	// Prevent go program from exiting.
	select {}
}

func showImg(in image.Image) {
	buf := &bytes.Buffer{}
	png.Encode(buf, in)

	dataURL := dataurl.New(buf.Bytes(), "image/png")
	byt, err := dataURL.MarshalText()
	if err != nil {
		showError("failed to encode data url", err)
		return
	}

	document := js.Global().Get("document")
	img := document.Call("createElement", "img")
	img.Set("src", string(byt))
	document.Get("body").Call("appendChild", img)

}
func showError(pfx string, err error) {
	// Adding an <h1> element in the HTML document
	document := js.Global().Get("document")
	p := document.Call("createElement", "h1")
	p.Set("innerHTML", pfx+": "+err.Error())
	document.Get("body").Call("appendChild", p)
}

func sayHello(this js.Value, inputs []js.Value) interface{} {
	firstArg := inputs[0].String()
	return "Hi " + firstArg + " from Go!"
}

func showSpr(this js.Value, arg []js.Value) interface{} {
	img, err := spr.DecodeOne(bytes.NewReader(sprBytes), arg[0].Int())
	if err != nil {
		showError("decoding a spr", err)
		select {}
	}

	showImg(img)

	return nil
}
