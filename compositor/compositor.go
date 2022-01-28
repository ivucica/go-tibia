// Package compositor paints a part of the map into an image.Image using the provided map data source and a things pack.
//
// It has support for compositing tiles, floors and the map itself, along with the lightmap overlay on top.
//
// BUG(ivucica): Light is not quite calculated per floor. Details below.
//
// 1) Lights on higher floors, if rendered, should affect lower floors
//    as well.
//
//    On OT map, see around x=111&y=108&bot=7&top=0: the light from
//    floor 6 is clearly visible until the character steps under the
//    house. Hence, lightmap should not enshroud floor 7 with config
//    x=111&y=108&bot=7&top=0 or x=111&y=108&bot=7&top=6, but it should
//    do so with x=111&y=108&bot=7&top=7.
// 2) Light sometimes penetrates from the lower floors. For instance,
//    fences on upper floors do get some light from the bottom floor,
//    but just one tile further away, the light is no longer visible.
// 3) While most light items seem to be behaving fine, the 'void'
//    (client item 100 on 8.54) is not emitting any light into the light
//    map, despite being marked as such in .dat (a brown color light
//    with strength).
package compositor

import (
	"image"
	"image/color"
	"image/draw"

	"badc0de.net/pkg/go-tibia/dat"
	"badc0de.net/pkg/go-tibia/gameworld"
	"badc0de.net/pkg/go-tibia/things"
	"github.com/golang/glog"
)

func compositeTile(t gameworld.MapTile, th *things.Things, img *image.RGBA, bottomRight image.Point, x, y uint16, floor uint8, tileW, tileH int) *Light {

	var light *Light

	idx := 0
	for item, err := t.GetItem(idx); err == nil; item, err = t.GetItem(idx) {
		thItem, err := th.Item(item.GetServerType(), 854)
		if err != nil {
			glog.Errorf("could not get item of type %d: %v", item.GetServerType(), err)
			continue
		}
		frame := thItem.ItemFrame(0, int(x), int(y), int(floor))

		dst := image.Rect(
			bottomRight.X-frame.Bounds().Size().X, bottomRight.Y-frame.Bounds().Size().Y,
			bottomRight.X, bottomRight.Y)

		draw.Draw(img, dst, frame, image.ZP, draw.Over)

		if thItem.LightInfo().Strength > 0 {
			light = &Light{
				Center: image.Pt(
					bottomRight.X-tileW/2,
					bottomRight.Y-tileH/2,
				),
				LightInfo: thItem.LightInfo(),
			}
		}

		idx++
	}

	idx = 0
	for creature, err := t.GetCreature(idx); err == nil; creature, err = t.GetCreature(idx) {
		// TODO: support item look for a creature
		glog.Infof("creature at %d %d %d (%08x) facing %v", x, y, floor, creature.GetID(), creature.GetDir())
		thCreature, err := th.Creature(creature.GetServerType(), 854)
		if err != nil {
			glog.Errorf("could not get creature of type %d: %v", creature.GetServerType(), err)
			continue
		}

		cols := creature.GetOutfitColors()
		glog.Infof("  -> look %d, colors %d %d %d %d", creature.GetServerType(), cols[0], cols[1], cols[2], cols[3])
		frame := thCreature.ColorizedCreatureFrame(0, creature.GetDir(), things.OutfitOverlayMask(0), []color.Color{cols[0], cols[1], cols[2], cols[3]})

		dst := image.Rect(
			bottomRight.X-frame.Bounds().Size().X, bottomRight.Y-frame.Bounds().Size().Y,
			bottomRight.X, bottomRight.Y)

		draw.Draw(img, dst, frame, image.ZP, draw.Over)

		// TODO: creature light?
		light = &Light{
			Center: image.Pt(
				bottomRight.X-tileW/2,
				bottomRight.Y-tileH/2,
			),
			LightInfo: dat.LightInfo{
				Color:    dat.DatasetColor(124),
				Strength: 3,
			},
		}
		idx++
	}

	return light
}

type Light struct {
	Center    image.Point
	LightInfo dat.LightInfo
}

var (
	//nightAmbient = color.RGBA{0, 0, 20, 240}
	//nightAmbient = color.RGBA{20, 20, 40, 240}
	nightAmbient      = dat.DatasetColor(0xD7)
	nightAmbientLevel = uint8(40)
	dayAmbient        = dat.DatasetColor(0xD7)
	dayAmbientLevel   = uint8(250)
)

func CompositeMap(m gameworld.MapDataSource, th *things.Things, x, y uint16, floorTop, floorBottom uint8, width, height int, tileW, tileH int) image.Image {
	fullSize := image.Rect(0, 0, width*tileW, height*tileH)
	img := image.NewRGBA(fullSize)

	var bottomRight image.Point

	ambientColor := nightAmbient
	ambientLevel := nightAmbientLevel

	for tz := int(floorBottom); tz >= int(floorTop); tz-- {
		off := int(tz - int(floorBottom))
		lights := []*Light{}
		floorImg := image.NewRGBA(fullSize)
		for ty := int(y) - off; ty < int(y)+height-off; ty++ {
			for tx := int(x) - off; tx < int(x)+width-off; tx++ {
				bottomRight.X = (tx - int(x) + 1 + off) * tileW
				bottomRight.Y = (ty - int(y) + 1 + off) * tileH

				t, err := m.GetMapTile(uint16(tx), uint16(ty), uint8(tz))
				if err != nil {
					glog.Errorf("error getting tile %d %d %d: %v", tx, ty, tz, err)
					continue
				}

				light := compositeTile(t, th, floorImg, bottomRight, uint16(tx), uint16(ty), uint8(tz), tileW, tileH)
				if light != nil {
					lights = append(lights, light)
				}

				// Add extra character for decoration.
				if (tx == int(x)+width/2) && (ty == int(y)+height/2) && tz == int(floorBottom) {
					/*
						light = &Light{
							Center: image.Point{width/2 * 32 + 32/2, height/2 * 32 + 32/2},
							LightInfo: dat.LightInfo{
								Color:    dat.DatasetColor(0xD7),
								Strength: 3,
							},
						}
						lights = append(lights, light)
					*/

					cr, err := th.CreatureWithClientID(128, 854)
					if err == nil {
						frame := cr.ColorizedCreatureFrame(0, 2, 0, []color.Color{things.OutfitColor(130), things.OutfitColor(90), things.OutfitColor(25), things.OutfitColor(130)})
						dst := image.Rect(
							bottomRight.X-frame.Bounds().Size().X, bottomRight.Y-frame.Bounds().Size().Y,
							bottomRight.X, bottomRight.Y)
						draw.Draw(floorImg, dst, frame, image.ZP, draw.Over)
					}
				}
			}
		}
		var overlay image.Image
		if true { // len(lights) > 0 {
			// debugging lightOverlay struct.
			if len(lights) > 0 && false {
				overlay = lightOverlayGen(width, height, tileW, tileH, lights[0], floorImg)
			} else {
				overlays := make([]image.Image, 0, len(lights))
				for idx, light := range lights {
					r, g, b, a := light.LightInfo.Color.RGBA()
					glog.Infof("LIGHT %d: center %+v, strength %d, color %d %d %d %d", idx, light.Center, light.LightInfo.Strength, r>>8, g>>8, b>>8, a>>8)
					overlay := lightOverlayGen(width, height, tileW, tileH, light, floorImg)
					overlays = append(overlays, overlay)
				}

				overlay = additiveOverlayGen(ambientColor, ambientLevel, width, height, tileW, tileH, overlays, floorImg)
			}
			draw.Draw(floorImg, fullSize, overlay, image.ZP, draw.Over)
		} else {
			overlay = compositeLightOverlayGen(width, height, tileW, tileH, lights, floorImg)
			draw.Draw(floorImg, fullSize, overlay, image.ZP, draw.Over)
		}

		draw.Draw(img, fullSize, floorImg, image.ZP, draw.Over)
	}

	//draw.Draw(img, fullSize, &image.Uniform{color.RGBA{255,255,255,255}}, image.ZP, draw.Src)

	return img
}
