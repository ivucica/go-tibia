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
