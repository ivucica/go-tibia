// Package compositor paints a part of the map into an image.Image using the provided map data source and a things pack.
//
// It has support for compositing tiles, floors and the map itself, along with the lightmap overlay on top.
//
// BUG(ivucica): Light is not quite calculated per floor. Details below.
//
//  1. Lights on higher floors, if rendered, should affect lower floors
//     as well.
//
//     On OT map, see around x=111&y=108&bot=7&top=0: the light from
//     floor 6 is clearly visible until the character steps under the
//     house. Hence, lightmap should not enshroud floor 7 with config
//     x=111&y=108&bot=7&top=0 or x=111&y=108&bot=7&top=6, but it should
//     do so with x=111&y=108&bot=7&top=7.
//
//  2. Light sometimes penetrates from the lower floors. For instance,
//     fences on upper floors do get some light from the bottom floor,
//     but just one tile further away, the light is no longer visible.
//
//  3. While most light items seem to be behaving fine, the 'void'
//     (client item 100 on 8.54) is not emitting any light into the light
//     map, despite being marked as such in .dat (a brown color light
//     with strength).
package compositor

import (
	"image"
	"image/draw"

	"badc0de.net/pkg/go-tibia/dat"
	"badc0de.net/pkg/go-tibia/gameworld"
	"badc0de.net/pkg/go-tibia/things"
)

type Light struct {
	Center    image.Point
	LightInfo dat.LightInfo
}

func CompositeMap(m gameworld.MapDataSource, th *things.Things, x, y uint16, floorTop, floorBottom uint8, width, height int, tileW, tileH int) image.Image {
	fullSize := image.Rect(0, 0, width*tileW, height*tileH)
	img := image.NewRGBA(fullSize)

	ambientColor, ambientLevel := m.GetAmbientLight()

	//func compositeFloor(m gameworld.MapDataSource, th *things.Things, x, y uint16, z uint8, off int, width, height int, tileW, tileH int, ambientColor color.Color, ambientLevel uint8) image.Image {
	for tz := int(floorBottom); tz >= int(floorTop); tz-- {
		off := int(tz - int(floorBottom))
		wantDecorativeCharacter := (tz == int(floorBottom))
		floorImg := compositeFloor(m, th, x, y, uint8(tz), off, width, height, tileW, tileH, ambientColor, ambientLevel, wantDecorativeCharacter)

		draw.Draw(img, fullSize, floorImg, image.ZP, draw.Over)
	}

	//draw.Draw(img, fullSize, &image.Uniform{color.RGBA{255,255,255,255}}, image.ZP, draw.Src)

	return img
}
