// +build js,wasm

package main

import (
	"bytes"
	"flag"
	"image"
	"image/png"
	"io"
	"log"
	"runtime"
	"runtime/debug"
	"syscall/js"

	"badc0de.net/pkg/go-tibia/compositor"
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

func (*console) Write(p []byte) (n int, err error) {
	js.Global().Get("window").Get("console").Call("log", js.ValueOf(string(p)))
	return len(p), nil
}

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
	js.Global().Set("loaderPromise", js.FuncOf(loaderPromise))
	js.Global().Set("addX", js.FuncOf(addX))
	js.Global().Set("addY", js.FuncOf(addY))
	js.Global().Set("subX", js.FuncOf(subX))
	js.Global().Set("subY", js.FuncOf(subY))

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

func loaderPromise(this js.Value, arg []js.Value) interface{} {
	// https://withblue.ink/2020/10/03/go-webassembly-http-requests-and-promises.html
	log.Println("constructing a loader promise")
	defer log.Println("constructed a loader promise")
	promiseConstructor := js.Global().Get("Promise")
	return promiseConstructor.New(js.FuncOf(loaderPromiseImp))
}

func loaderPromiseImp(this js.Value, arg []js.Value) interface{} {
	resolve := arg[0]
	reject := arg[1]

	log.Print("starting the loader promise")
	defer log.Println("started the loader promise")
	go loaderImp(resolve, reject)
	return nil
}

func failPromise(err error, pfx string, reject js.Value) {
	errorConstructor := js.Global().Get("Error")
	errorObject := errorConstructor.New(err.Error())

	log.Printf("failing the promise: %v", err)
	reject.Invoke(errorObject)
}

func loaderImp(resolve, reject js.Value) {
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
	debug.FreeOSMemory()

	resolve.Invoke(nil)
}

func showMap(this js.Value, arg []js.Value) interface{} {

	//log.Println("loaded, now compositing")
	m := globalMap
	t := globalThings
	img := compositor.CompositeMap(m, t, tx, ty, ttop, tbot, tw, th, 32, 32)
	//log.Println("composited")

	showImg(img, "map", true)

	return nil
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
