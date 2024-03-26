package compositor

import (
	"image"
	"image/color"
	"image/draw"
	"math"

	"github.com/golang/glog"
)


type compositeLightOverlay struct {
	ambientColor color.Color // most often dat.DatasetColor
	ambientLevel uint8

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

func (o *compositeLightOverlay) At(x, y int) color.Color {
	if _, _, _, a := o.base.At(x, y).RGBA(); a == 0 {
		return color.RGBA{0, 0, 0, 0}
	}

	//var c color.Color
	//c = nightAmbient

	aR, aG, aB, aA := o.ambientColor.RGBA()

	brightness := float64(o.ambientLevel) / 255.0
	aR = uint32(float64(aR) * brightness)
	aG = uint32(float64(aG) * brightness)
	aB = uint32(float64(aB) * brightness)
	aA = uint32(float64(aA) * brightness)
	c := color.RGBA{
		R: uint8(aR >> 8),
		G: uint8(aG >> 8),
		B: uint8(aB >> 8),
		A: uint8(aA >> 8),
	}

	for _, l := range o.lights {
		// Distance to the light, in the form of fraction of overlay's width and height.
		dX := float64(l.Center.X-x) / float64(o.bounds.Dx())
		dY := float64(l.Center.Y-y) / float64(o.bounds.Dy())

		// Radius of effect, as fraction of image width.
		// TODO: height is commonly different than width.
		r := float64(l.LightInfo.Strength-1) * 32 / float64(o.bounds.Dx())

		// A bad formula for light fall-off.
		f := float64((r*r)-(dX*dX+dY*dY)) + 32/float64(o.bounds.Dx())
		if f < 0 {
			// too far
			continue
		}

		if f > 1 {
			f = 1
		}

		f = 1 - f
		f = math.Pow(f, 10)

		//c = l.LightInfo.Color

		//c.R = uint8(float64(c.R)+64*(1.0-f))
		//c.G = uint8(float64(c.G)+64*(1.0-f))
		//c.B = uint8(float64(c.B)+64*(1.0-f))

		c.A = uint8(float64(c.A) * f)
	}
	return c
}

func compositeLightOverlayGen(width, height int, tileW, tileH int, lights []*Light, ambientColor color.Color, ambientLevel uint8, base image.Image) image.Image {
	fullSize := image.Rect(0, 0, width*tileW, height*tileH)
	img := image.NewRGBA(fullSize)

	//if len(lights) == 0 {
	//glog.Infof("no lights")
	//draw.Draw(img, fullSize, &image.Uniform{nightOverlay}, image.ZP, draw.Src)
	//} else {
	glog.Infof("%d lights", len(lights))
	draw.Draw(img, fullSize, &compositeLightOverlay{bounds: fullSize, lights: lights, ambientColor: ambientColor, ambientLevel: ambientLevel, base: base}, image.ZP, draw.Src)
	//}

	return img
}
