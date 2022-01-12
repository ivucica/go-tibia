// +build js,wasm

package main

import (
	"bytes"
	"image"
	"image/png"
	"io"
	"syscall/js"
	
	"badc0de.net/pkg/go-tibia/spr"
	"badc0de.net/pkg/go-tibia/paths"
	"github.com/vincent-petithory/dataurl"
)

// Contains single seekable buffer wrapped with a bytes.NewReader.
//
// TODO(ivucica): Hypothetically different goroutines could seek around it differently.
var sprReaderSeekerCloser interface{io.ReadSeeker; io.Closer}

func main() {

	r, err := paths.Open("Tibia.spr")
	if err != nil {
		showError("opening Tibia.spr", err)
		select {}
	}

	sprReaderSeekerCloser = r
	
	img, err := spr.DecodeOne(r, 500)
	if err != nil {
		showError("decoding a spr", err)
		select {}
	}
	r.Close()

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
	_, err := sprReaderSeekerCloser.Seek(0, io.SeekStart)
	if err != nil {
		showError("seeking spr to start", err)
	}
	img, err := spr.DecodeOne(sprReaderSeekerCloser, arg[0].Int())
	if err != nil {
		showError("decoding a spr", err)
		select {}
	}

	showImg(img)

	return nil
}
