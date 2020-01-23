package gameworld

import (
	"image"
	"image/draw"

	"badc0de.net/pkg/go-tibia/things"
	"github.com/golang/glog"
)

func compositeTile(t MapTile, th *things.Things, img *image.RGBA, bottomRight image.Point, x, y uint16, floor uint8) {
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
		idx++
	}
}

func CompositeMap(m MapDataSource, th *things.Things, x, y uint16, floorTop, floorBottom uint8, width, height int, tileW, tileH int) image.Image {
	img := image.NewRGBA(image.Rect(0, 0, width*tileW, height*tileH))

	var bottomRight image.Point

	for tz := int(floorBottom); tz >= int(floorTop); tz-- {
		off := int(tz - int(floorBottom))
		for ty := int(y)-off; ty < int(y)+height-off; ty++ {
			for tx := int(x)-off; tx < int(x)+width-off; tx++ {
				bottomRight.X = (tx - int(x) + 1 + off) * tileW
				bottomRight.Y = (ty - int(y) + 1 + off) * tileH

				t, err := m.GetMapTile(uint16(tx), uint16(ty), uint8(tz))
				if err != nil {
					glog.Errorf("error getting tile %d %d %d: %v", tx, ty, tz, err)
					continue
				}
				compositeTile(t, th, img, bottomRight, uint16(tx), uint16(ty), uint8(tz))
			}
		}
	}
	return img
}
