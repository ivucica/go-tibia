package gameworld

import (
	"image"
	"image/color"
	"image/draw"
	"math"

	"badc0de.net/pkg/go-tibia/v1/dat"
	"badc0de.net/pkg/go-tibia/v1/things"
	"github.com/golang/glog"
)

func compositeTile(t MapTile, th *things.Things, img *image.RGBA, bottomRight image.Point, x, y uint16, floor uint8, tileW, tileH int) *Light {

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

	return light
}

type Light struct {
	Center    image.Point
	LightInfo dat.LightInfo
}

type compositeLightOverlay struct {
	lights []*Light
	bounds image.Rectangle
	base   image.Image
}

func (*compositeLightOverlay) ColorModel() color.Model {
	return color.RGBAModel
}

func (o *compositeLightOverlay) Bounds() image.Rectangle {
	return o.bounds
}

var (
	nightOverlay = color.RGBA{0, 0, 20, 240}
)

func (o *compositeLightOverlay) At(x, y int) color.Color {
	if _, _, _, a := o.base.At(x, y).RGBA(); a == 0 {
		return color.RGBA{0, 0, 0, 0}
	}

	c := nightOverlay
	for _, l := range o.lights {
		dX := float64(l.Center.X-x) / float64(o.bounds.Dx())
		dY := float64(l.Center.Y-y) / float64(o.bounds.Dy())
		r := float64(l.LightInfo.Strength-1) * 32 / float64(o.bounds.Dx())
		f := float64((r*r)-(dX*dX+dY*dY)) + 32/float64(o.bounds.Dx())
		if f < 0 {
			// too far
			continue
		}

		//glog.Infof("%g", f)
		if f > 1 {
			f = 1
		}

		f = 1 - f
		f = math.Pow(f, 10)

		//c.R = uint8(float64(c.R)+64*(1.0-f))
		//c.G = uint8(float64(c.G)+64*(1.0-f))
		//c.B = uint8(float64(c.B)+64*(1.0-f))
		c.A = uint8(float64(c.A) * f)
	}
	return c
}

func compositeLightOverlayGen(width, height int, tileW, tileH int, lights []*Light, base image.Image) image.Image {
	fullSize := image.Rect(0, 0, width*tileW, height*tileH)
	img := image.NewRGBA(fullSize)

	//if len(lights) == 0 {
	//glog.Infof("no lights")
	//draw.Draw(img, fullSize, &image.Uniform{nightOverlay}, image.ZP, draw.Src)
	//} else {
	glog.Infof("%d lights", len(lights))
	draw.Draw(img, fullSize, &compositeLightOverlay{bounds: fullSize, lights: lights, base: base}, image.ZP, draw.Src)
	//}

	return img
}

func CompositeMap(m MapDataSource, th *things.Things, x, y uint16, floorTop, floorBottom uint8, width, height int, tileW, tileH int) image.Image {
	fullSize := image.Rect(0, 0, width*tileW, height*tileH)
	img := image.NewRGBA(fullSize)

	var bottomRight image.Point

	for tz := int(floorBottom); tz >= int(floorTop); tz-- {
		off := int(tz - int(floorBottom))
		lights := []*Light{}
		floor := image.NewRGBA(fullSize)
		for ty := int(y) - off; ty < int(y)+height-off; ty++ {
			for tx := int(x) - off; tx < int(x)+width-off; tx++ {
				bottomRight.X = (tx - int(x) + 1 + off) * tileW
				bottomRight.Y = (ty - int(y) + 1 + off) * tileH

				t, err := m.GetMapTile(uint16(tx), uint16(ty), uint8(tz))
				if err != nil {
					glog.Errorf("error getting tile %d %d %d: %v", tx, ty, tz, err)
					continue
				}

				light := compositeTile(t, th, floor, bottomRight, uint16(tx), uint16(ty), uint8(tz), tileW, tileH)
				if light != nil {
					lights = append(lights, light)
				}

				if (tx == int(x)+width/2) && (ty == int(y)+height/2) && tz == int(floorBottom) {
					cr, err := th.CreatureWithClientID(128, 854)
					if err == nil {
						frame := cr.ColorizedCreatureFrame(0, 2, 0, []color.Color{things.OutfitColor(130), things.OutfitColor(90), things.OutfitColor(25), things.OutfitColor(130)})
						dst := image.Rect(
							bottomRight.X-frame.Bounds().Size().X, bottomRight.Y-frame.Bounds().Size().Y,
							bottomRight.X, bottomRight.Y)
						draw.Draw(floor, dst, frame, image.ZP, draw.Over)
					}
				}
			}
		}
		overlay := compositeLightOverlayGen(width, height, tileW, tileH, lights, floor)
		draw.Draw(floor, fullSize, overlay, image.ZP, draw.Over)

		draw.Draw(img, fullSize, floor, image.ZP, draw.Over)
	}

	//draw.Draw(img, fullSize, &image.Uniform{color.RGBA{255,255,255,255}}, image.ZP, draw.Src)

	return img
}
