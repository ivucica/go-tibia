//go:build (js && wasm) || tinygo.wasm
// +build js,wasm tinygo.wasm

package main

import (
	"bytes"
	"context"
	"flag"
	"image"
	"image/png"
	"io"
	"log"
	"runtime"

	"badc0de.net/pkg/go-tibia/compositor/dom"
	"badc0de.net/pkg/go-tibia/gameworld"
	"badc0de.net/pkg/go-tibia/otb/map"
	"badc0de.net/pkg/go-tibia/paths"
	"badc0de.net/pkg/go-tibia/spr"
	"badc0de.net/pkg/go-tibia/things"
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

var (
	globalThings *things.Things
	globalMap    gameworld.MapDataSource

	// TODO
	tx, ty     uint16
	tbot, ttop uint8
	tw, th     int
)

const mapPath = ":test:"

func main() {

	tx = 84
	ty = 83 // 84
	tbot = 7
	ttop = 0
	tw = 18
	th = 14

	flag.Set("logtostderr", "true")
	flag.Parse()
	log.SetOutput(&console{})

	log.Println("opening tibia.spr")
	r, err := paths.Open("Tibia.spr")
	if err != nil {
		showError("opening Tibia.spr", err)
		persistIfBrowser() // does not exit on browser
		return
	}

	sprReaderSeekerCloser = r

	log.Println("decoding 500")
	img, err := spr.DecodeOne(r, 500)
	if err != nil {
		showError("decoding a spr", err)
		persistIfBrowser() // does not exit on browser
		return
	}
	r.Close()

	showImg(img, "sprites", false)

	//2. Exposing go functions/values in javascript variables.
	jsGlobalInjectAPI()

	// Prevent go program from exiting.
	persistIfBrowser()
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

	document := jsGlobal().Get("document")
	img := document.Call("createElement", "img")
	img.Set("src", string(byt))

	showDOM(img, parentElementID, replace)
}

func showError(pfx string, err error) {
	log.Printf("[E] %s: %v", pfx, err)

	// Adding an <h1> element in the HTML document
	document := jsGlobal().Get("document")
	p := document.Call("createElement", "h1")
	p.Set("innerHTML", pfx+": "+err.Error())
	document.Get("body").Call("appendChild", p)
}

func sayHello(this jsValue, inputs []jsValue) interface{} {
	firstArg := inputs[0].String()
	return "Hi " + firstArg + " from Go!"
}

//export ExportedSayHello
func ExportedSayHello(this jsValue, inputs []jsValue) interface{} {
	// Exported version of the function usable in gotwebfe-runner.
	return sayHello(this, inputs)
}

func showSpr(this jsValue, arg []jsValue) interface{} {
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

func loaderPromiseImp(this jsValue, arg []jsValue) interface{} {
	resolve := arg[0]
	reject := arg[1]

	log.Print("starting the loader promise")
	defer log.Println("started the loader promise")
	go loaderImp(resolve, reject)
	return nil
}

func failPromise(err error, pfx string, reject jsValue) {
	errorConstructor := jsGlobal().Get("Error")
	errorObject := errorConstructor.New(err.Error())

	log.Printf("failing the promise: %v", err)
	reject.Invoke(errorObject)
}

func loaderImp(resolve, reject jsValue) {
	defer log.Printf("loader promise exiting")
	log.Println("loading all from default paths")
	t, err := full.FromDefaultPaths(true)
	if err != nil {
		failPromise(err, "loading required files from default paths", reject)
		return
	}
	log.Println("loaded things!")

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

	var m gameworld.MapDataSource
	if mapPath == ":test:" {
		m = gameworld.NewMapDataSource()
	} else {
		f, err := paths.Open(mapPath)
		if err != nil {
			failPromise(err, "opening map file", reject)
			return
		}
		m, err = otbm.New(f, t)
		if err != nil {
			failPromise(err, "reading map file", reject)
			return
		}
		f.Close()
	}

	globalMap = m
	globalThings = t

	log.Printf("loaded global map %p and things %p", m, t)

	runtime.GC()
	debugFreeOSMemory()

	resolve.Invoke(nil)
}

func showMap(this jsValue, arg []jsValue) interface{} {
	ctx := context.TODO()
	window := jsGlobal()

	//log.Println("loaded, now compositing")
	m := globalMap
	t := globalThings

	if mp, err := dom.CompositeMapToDOM(ctx, window, m, t, tx, ty, ttop, tbot, tw, th, 32, 32, useImgBased, useWellKnownUrls); err == nil {
		showDOM(mp, "map", true)
	} else {
		showError("showMap", err)
	}
	//log.Println("composited")

	// TODO(ivucica): clear images from globalThings
	runtime.GC()
	debugFreeOSMemory()

	return nil
}
