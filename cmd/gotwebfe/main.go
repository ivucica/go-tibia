// +build js,wasm

package main

import (
	"bytes"
	"image"
	"image/png"
	"io"
	"log"
	"syscall/js"

	"badc0de.net/pkg/go-tibia/gameworld"
	"badc0de.net/pkg/go-tibia/otb/map"
	"badc0de.net/pkg/go-tibia/paths"
	"badc0de.net/pkg/go-tibia/spr"
	"badc0de.net/pkg/go-tibia/things/full"
	"badc0de.net/pkg/go-tibia/xmls"
	"github.com/vincent-petithory/dataurl"
)

// Contains single seekable buffer wrapped with a bytes.NewReader.
//
// TODO(ivucica): Hypothetically different goroutines could seek around it differently.
var sprReaderSeekerCloser interface {
	io.ReadSeeker
	io.Closer
}

type console struct{}

func (*console) Write(p []byte) (n int, err error) {
	js.Global().Get("window").Get("console").Call("log", js.ValueOf(string(p)))
	return len(p), nil
}

func main() {

	log.SetOutput(&console{})

	log.Println("opening tibia.spr")
	r, err := paths.Open("Tibia.spr")
	if err != nil {
		showError("opening Tibia.spr", err)
		select {}
	}

	sprReaderSeekerCloser = r

	log.Println("decoding 500")
	img, err := spr.DecodeOne(r, 500)
	if err != nil {
		showError("decoding a spr", err)
		select {}
	}
	r.Close()

	showImg(img, "sprites", false)

	//2. Exposing go functions/values in javascript variables.
	js.Global().Set("goVar", "I am a variable set from Go")
	js.Global().Set("sayHello", js.FuncOf(sayHello))

	js.Global().Set("showSpr", js.FuncOf(showSpr))
	js.Global().Set("showMap", js.FuncOf(showMap))

	// Prevent go program from exiting.
	select {}
}

func showImg(in image.Image, parentElementID string, replace bool) {
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
	parent := document.Call("getElementById", parentElementID)
	if parent.IsNull() || parent.IsUndefined() {
		log.Printf("[W] showImg: appending to body, because %q cannot be found", parentElementID)
		parent = document.Get("body")
	}
	if replace {
		log.Printf("[W] showImg: replace not supported yet, just appending")
	}
	parent.Call("appendChild", img)

}
func showError(pfx string, err error) {
	log.Printf("[E] %s: %v", pfx, err)
	
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
		return nil
	}
	img, err := spr.DecodeOne(sprReaderSeekerCloser, arg[0].Int())
	if err != nil {
		showError("decoding a spr", err)
		return nil
	}

	showImg(img, "sprites", false)

	return nil
}

func showMap(this js.Value, arg []js.Value) interface{} {
	log.Println("loading all from default paths")
	t, err := full.FromDefaultPaths(true)
	if err != nil {
		//showError("loading required files from default paths", err)
		return nil
	}

	log.Println("opening outfits.xml")
	var outfitsXML *xmls.Outfits
	f, err := paths.Open("outfits.xml")
	if err != nil {
		//showError("could not open outfits xml", err)
		outfitsXML = &xmls.Outfits{}
	} else {
		outfits, err := xmls.ReadOutfits(f)
		if err != nil {
			//showError("could not parse outfits xml", err)
			outfitsXML = &xmls.Outfits{}
		} else {
			outfitsXML = &outfits
		}
	}
	f.Close()

	// TODO: use outfitsXML
	outfitsXML = outfitsXML

	// TODO
	mapPath := ":test:" // "map.otbm"
	var tx, ty uint16
	var tbot, ttop uint8
	var tw, th int

	tx = 84
	ty = 84
	tbot = 7
	ttop = 0
	tw = 18
	th = 14

	var m gameworld.MapDataSource
	if mapPath == ":test:" {
		m = gameworld.NewMapDataSource()
	} else {
		f, err := paths.Open(mapPath)
		if err != nil {
			//showError("opening map file", err)
			return nil
		}
		m, err = otbm.New(f, t)
		if err != nil {
			//showError("reading map file", err)
			return nil
		}
		f.Close()
	}
	//log.Println("loaded, now compositing")
	img := gameworld.CompositeMap(m, t, tx, ty, ttop, tbot, tw, th, 32, 32)
	//log.Println("composited")

	showImg(img, "map", true)

	return nil
}
