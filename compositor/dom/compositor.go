//go:build js && wasm && !tinygo.wasm
// +build js,wasm,!tinygo.wasm

package dom

import (
	"bytes"
	"context"
	"fmt"
	"image/color"
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
	mapDiv.Get("style").Set("width", fmt.Sprintf("%dpx", width*tileW))
	mapDiv.Get("style").Set("height", fmt.Sprintf("%dpx", height*tileH))

	// Contains all floorDivOuters.
	allFloorDivs := []js.Value{}

	floorW := width * tileW
	floorH := height * tileH
	floorWStr := fmt.Sprintf("%dpx", floorW)
	floorHStr := fmt.Sprintf("%dpx", floorH)

	for tz := int(floorBottom); tz >= int(floorTop); tz-- {
		allTileDivs := []js.Value{}

		for ty := int(y); ty < int(y)+height; ty++ {
			for tx := int(x); tx < int(x)+width; tx++ {
				t, err := m.GetMapTile(uint16(tx), uint16(ty), uint8(tz))
				if err != nil {
					return js.Null(), fmt.Errorf("error getting tile %d %d %d: %v", tx, ty, tz, err)
				}

				// Positioning tiles relative to position of the div itself.
				//
				// Tiles are placed inside a position:relative div of the floor itself, so position:absolute is fine.
				tileLeft := (tx - int(x)) * tileW
				tileTop := (ty - int(y)) * tileH

				itemDivs := []js.Value{}
				creatureDivs := []js.Value{}

				idx := 0
				for item, err := t.GetItem(idx); err == nil; item, err = t.GetItem(idx) {
					thItem, err := th.Item(item.GetServerType(), 854)
					if err != nil {
						return js.Null(), fmt.Errorf("could not get item of type %d: %v", item.GetServerType(), err)
					}
					frame := thItem.ItemFrame(0, int(tx), int(ty), int(tz))

					// Positioning items relative to the tile itself.
					//
					// Because items and creatures can be larger than tileW/tileH, and the origin point is bottom-right corner, we use that to position the images.
					itemDiv := document.Call("createElement", "div")
					itemDiv.Get("style").Set("position", "absolute")
					itemDiv.Get("style").Set("right", fmt.Sprintf("%dpx", 0))
					itemDiv.Get("style").Set("bottom", fmt.Sprintf("%dpx", 0))
					itemDiv.Get("style").Set("width", fmt.Sprintf("%dpx", tileW))  // TODO: item size does NOT match tile size
					itemDiv.Get("style").Set("height", fmt.Sprintf("%dpx", tileH)) // TODO: item size does NOT match tile size

					// TODO: offer option to use pre-prepared PNGs or other image formats, which browsers would like and could cache.
					buf := &bytes.Buffer{}
					png.Encode(buf, frame)
					dataURL := dataurl.New(buf.Bytes(), "image/png")
					byt, err := dataURL.MarshalText()
					if err != nil {
						return js.Null(), fmt.Errorf("failed to encode data url: %w", err)
					}
					itemDiv.Get("style").Set("backgroundImage", fmt.Sprintf("url('%s')", string(byt)))

					itemDivs = append(itemDivs, itemDiv)
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

					// Positioning creatures relative to the tile itself.
					//
					// Because items and creatures can be larger than tileW/tileH, and the origin point is bottom-right corner, we use that to position the images.
					creatureDiv := document.Call("createElement", "div")
					creatureDiv.Get("style").Set("position", "absolute")
					creatureDiv.Get("style").Set("right", fmt.Sprintf("%dpx", 0))
					creatureDiv.Get("style").Set("bottom", fmt.Sprintf("%dpx", 0))
					creatureDiv.Get("style").Set("width", fmt.Sprintf("%dpx", tileW))  // TODO: creature size does NOT match tile size
					creatureDiv.Get("style").Set("height", fmt.Sprintf("%dpx", tileH)) // TODO: creature size does NOT match tile size

					// TODO: offer option to use pre-prepared PNGs or other image formats, which browsers would like and could cache.
					buf := &bytes.Buffer{}
					png.Encode(buf, frame)
					dataURL := dataurl.New(buf.Bytes(), "image/png")
					byt, err := dataURL.MarshalText()
					if err != nil {
						return js.Null(), fmt.Errorf("failed to encode data url: %w", err)
					}
					creatureDiv.Get("style").Set("backgroundImage", fmt.Sprintf("url('%s')", string(byt)))

					creatureDivs = append(creatureDivs, creatureDiv)
					idx++
				}

				if len(itemDivs) > 0 || len(creatureDivs) > 0 {
					tileDiv := document.Call("createElement", "div")
					tileDiv.Get("style").Set("position", "absolute")
					tileDiv.Get("style").Set("left", fmt.Sprintf("%dpx", tileLeft))
					tileDiv.Get("style").Set("top", fmt.Sprintf("%dpx", tileTop))
					tileDiv.Get("style").Set("width", fmt.Sprintf("%dpx", tileW))
					tileDiv.Get("style").Set("height", fmt.Sprintf("%dpx", tileH))
					for _, itemDiv := range itemDivs {
						tileDiv.Call("appendChild", itemDiv)
					}
					for _, creatureDiv := range creatureDivs {
						tileDiv.Call("appendChild", creatureDiv)
					}

					allTileDivs = append(allTileDivs, tileDiv)
				}
			}
		}

		if len(allTileDivs) > 0 {
			floorDivOuter := document.Call("createElement", "div")
			floorDivOuter.Get("style").Set("position", "absolute")
			floorDivOuter.Get("style").Set("width", floorWStr)
			floorDivOuter.Get("style").Set("height", floorHStr)
			floorDivOuter.Get("style").Set("top", "0")
			floorDivOuter.Get("style").Set("left", "0")

			floorDiv := document.Call("createElement", "div")
			floorDiv.Get("style").Set("position", "relative")
			floorDiv.Get("style").Set("width", floorWStr)
			floorDiv.Get("style").Set("height", floorHStr)

			for _, tileDiv := range allTileDivs {
				floorDiv.Call("appendChild", tileDiv)
			}

			floorDivOuter.Call("appendChild", floorDiv)

			// Performing this as the last action would group redraws together...
			// mapDiv.Call("appendChild", floorDivOuter)
			// but it is even better to collect them for later.
			allFloorDivs = append(allFloorDivs, floorDivOuter)
		}
	}

	for _, floorDivOuter := range allFloorDivs {
		mapDiv.Call("appendChild", floorDivOuter)
	}

	// TODO: a fast-to-generate lightmap, possibly prepared per-tile; or better, a low-res image stretched as an overlay per-floor.

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
