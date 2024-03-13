//go:build js && wasm
// +build js,wasm

package dom

import (
	"bytes"
	"context"
	"fmt"
	"image/png"
	"syscall/js"

	"badc0de.net/pkg/go-tibia/compositor"
	"badc0de.net/pkg/go-tibia/gameworld"
	"badc0de.net/pkg/go-tibia/things"

	"github.com/vincent-petithory/dataurl"
)

// CompositeMapToDOM returns a DOM object (a js.Value) which represents an HTML element displaying the requested slice of the gameworld in a way that makes sense to the player.
//
// Currently, this is just the img.
//
// Argument window should be the value of js.Global(), either 'window' or 'global', since it should offer us a 'document' that can then be used to construct the returned tree.
func CompositeMapToDOM(ctx context.Context, window js.Value, m gameworld.MapDataSource, th *things.Things, x, y uint16, floorTop, floorBottom uint8, width, height int, tileW, tileH int) (js.Value, error) {

	in := compositor.CompositeMap(m, th, x, y, floorTop, floorBottom, width, height, tileW, tileH)

	buf := &bytes.Buffer{}
	png.Encode(buf, in)

	dataURL := dataurl.New(buf.Bytes(), "image/png")
	byt, err := dataURL.MarshalText()
	if err != nil {
		return js.Null(), fmt.Errorf("failed to encode data url: %w", err)
	}

	document := window.Get("document")
	img := document.Call("createElement", "img")
	img.Set("src", string(byt))

	return img, nil
}
