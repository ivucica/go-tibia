package compositor

import (
	"image"
	"image/color"
	"image/draw"

	"badc0de.net/pkg/go-tibia/gameworld"
	"badc0de.net/pkg/go-tibia/things"
	"github.com/golang/glog"
)

func compositeFloor(m gameworld.MapDataSource, th *things.Things, x, y uint16, z uint8, off int, width, height int, tileW, tileH int, ambientColor color.Color, ambientLevel uint8, decorativeCharacter_Temp bool) image.Image {
	fullSize := image.Rect(0, 0, width*tileW, height*tileH)

	var bottomRight image.Point

	tz := z // renaming

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
			if (tx == int(x)+width/2) && (ty == int(y)+height/2) && decorativeCharacter_Temp {
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
		overlay = compositeLightOverlayGen(width, height, tileW, tileH, lights, ambientColor, ambientLevel, floorImg)
		draw.Draw(floorImg, fullSize, overlay, image.ZP, draw.Over)
	}

	return floorImg
}
