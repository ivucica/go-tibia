//go:build js && wasm && !tinygo.wasm
// +build js,wasm,!tinygo.wasm

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

// compositeMapToDOMAsDataURL provides a single <img> with a data URL src in it; a png.
func compositeMapToDOMAsDataURL(ctx context.Context, window js.Value, m gameworld.MapDataSource, th *things.Things, x, y uint16, floorTop, floorBottom uint8, width, height int, tileW, tileH int) (js.Value, error) {
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

// compositeMapToDOMAsManyDIVs creates many imgs.
//
// BUG: no caching, no attempt to use well-known URLs that would be provided by service worker (/item/... and usch).
func compositeMapToDOMAsManyDIVs(ctx context.Context, window js.Value, m gameworld.MapDataSource, th *things.Things, x, y uint16, floorTop, floorBottom uint8, width, height int, tileW, tileH int) (js.Value, error) {
	document := window.Get("document")
	mapDiv := document.Call("createElement", "div")
	mapDiv.Get("style").Set("position", "relative")

	for tz := int(floorBottom); tz >= int(floorTop); tz-- {
		floorDiv := document.Call("createElement", "div")
		floorDiv.Get("style").Set("position", "relative")
		floorDiv.Get("style").Set("width", fmt.Sprintf("%dpx", width*tileW))
		floorDiv.Get("style").Set("height", fmt.Sprintf("%dpx", height*tileH))

		for ty := int(y); ty < int(y)+height; ty++ {
			for tx := int(x); tx < int(x)+width; tx++ {
				t, err := m.GetMapTile(uint16(tx), uint16(ty), uint8(tz))
				if err != nil {
					return js.Null(), fmt.Errorf("error getting tile %d %d %d: %v", tx, ty, tz, err)
				}

				tileDiv := document.Call("createElement", "div")
				tileDiv.Get("style").Set("position", "absolute")
				tileDiv.Get("style").Set("left", fmt.Sprintf("%dpx", (tx-int(x))*tileW))
				tileDiv.Get("style").Set("top", fmt.Sprintf("%dpx", (ty-int(y))*tileH))
				tileDiv.Get("style").Set("width", fmt.Sprintf("%dpx", tileW))
				tileDiv.Get("style").Set("height", fmt.Sprintf("%dpx", tileH))

				idx := 0
				for item, err := t.GetItem(idx); err == nil; item, err = t.GetItem(idx) {
					thItem, err := th.Item(item.GetServerType(), 854)
					if err != nil {
						return js.Null(), fmt.Errorf("could not get item of type %d: %v", item.GetServerType(), err)
					}
					frame := thItem.ItemFrame(0, int(tx), int(ty), int(tz))

					itemDiv := document.Call("createElement", "div")
					itemDiv.Get("style").Set("position", "absolute")
					itemDiv.Get("style").Set("left", fmt.Sprintf("%dpx", 0))
					itemDiv.Get("style").Set("top", fmt.Sprintf("%dpx", 0))
					itemDiv.Get("style").Set("width", fmt.Sprintf("%dpx", tileW))
					itemDiv.Get("style").Set("height", fmt.Sprintf("%dpx", tileH))

					buf := &bytes.Buffer{}
					png.Encode(buf, frame)
					dataURL := dataurl.New(buf.Bytes(), "image/png")
					byt, err := dataURL.MarshalText()
					if err != nil {
						return js.Null(), fmt.Errorf("failed to encode data url: %w", err)
					}
					itemDiv.Get("style").Set("backgroundImage", fmt.Sprintf("url('%s')", string(byt)))

					tileDiv.Call("appendChild", itemDiv)
					idx++
				}

				idx = 0
				for creature, err := t.GetCreature(idx); err == nil; creature, err = t.GetCreature(idx) {
					thCreature, err := th.Creature(creature.GetServerType(), 854)
					if err != nil {
						return js.Null(), fmt.Errorf("could not get creature of type %d: %v", creature.GetServerType(), err)
					}

					cols := creature.GetOutfitColors()
					frame := thCreature.ColorizedCreatureFrame(0, creature.GetDir(), things.OutfitOverlayMask(0), []color.Color{cols[0], cols[1], cols[2], cols[3]})

					creatureDiv := document.Call("createElement", "div")
					creatureDiv.Get("style").Set("position", "absolute")
					creatureDiv.Get("style").Set("left", fmt.Sprintf("%dpx", 0))
					creatureDiv.Get("style").Set("top", fmt.Sprintf("%dpx", 0))
					creatureDiv.Get("style").Set("width", fmt.Sprintf("%dpx", tileW))
					creatureDiv.Get("style").Set("height", fmt.Sprintf("%dpx", tileH))

					buf := &bytes.Buffer{}
					png.Encode(buf, frame)
					dataURL := dataurl.New(buf.Bytes(), "image/png")
					byt, err := dataURL.MarshalText()
					if err != nil {
						return js.Null(), fmt.Errorf("failed to encode data url: %w", err)
					}
					creatureDiv.Get("style").Set("backgroundImage", fmt.Sprintf("url('%s')", string(byt)))

					tileDiv.Call("appendChild", creatureDiv)
					idx++
				}

				floorDiv.Call("appendChild", tileDiv)
			}
		}

		mapDiv.Call("appendChild", floorDiv)
	}

	return mapDiv, nil
}

// CompositeMapToDOM returns a DOM object (a js.Value) which represents an HTML element displaying the requested slice of the gameworld in a way that makes sense to the player.
//
// Currently, this is just the img.
//
// Argument window should be the value of js.Global(), either 'window' or 'global', since it should offer us a 'document' that can then be used to construct the returned tree.
func CompositeMapToDOM(ctx context.Context, window js.Value, m gameworld.MapDataSource, th *things.Things, x, y uint16, floorTop, floorBottom uint8, width, height int, tileW, tileH int) (js.Value, error) {
	return compositeMapToDOMAsManyDIVs(ctx, window, m, th, x, y, floorTop, floorBottom, width, height, tileW, tileH)
}
