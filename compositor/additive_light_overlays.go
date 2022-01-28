package compositor

import (
	"image"
	"image/color"
	"image/draw"
	"math"

	"github.com/golang/glog"
)

type lightOverlay struct {
	light  *Light
	bounds image.Rectangle
	base   image.Image
}

func (*lightOverlay) ColorModel() color.Model {
	return color.RGBAModel
}

func (o *lightOverlay) Bounds() image.Rectangle {
	return o.bounds
}

func (o *lightOverlay) At(x, y int) color.Color {
	if _, _, _, a := o.base.At(x, y).RGBA(); a == 0 {
		return color.RGBA{0, 0, 0, 0}
	}

	// Start with black, containing alpha-premultiplied RGBA.
	// c.{R,G,B,A} are uint8.
	c := color.RGBA{0, 0, 0, 0}

	// Pick the light.
	l := o.light

	//////
	// Radius of effect.
	//radius := float64(l.LightInfo.Strength-1) * 32
	//bonusRadiusMultiplier := float64(16)

	radius := float64(l.LightInfo.Strength) * 16
	bonusRadiusMultiplier := float64(1)

	//radius := float64(l.LightInfo.Strength-1)
	radius *= bonusRadiusMultiplier

	//////
	// Vector of distance to the light.
	dX := float64(l.Center.X - x)
	dY := float64(l.Center.Y - y)

	// Length of the vector.
	dSquare := dX*dX + dY*dY
	if dSquare < 0 {
		dSquare = 0
	}
	//const falloffGrant = 2.0 // Extra grant for falloff (let's not have too strong cutoff).
	//if dSquare > radius*radius*falloffGrant {
	// Very far out of range. Skip expensive operations like square root.
	//	return c
	//}
	d := math.Sqrt(dSquare)

	//////
	// Calculate influence.
	var influence float64
	if false {
		// Clamped attenuation.
		attenuation := radius / dSquare
		if attenuation < 0 {
			attenuation = 0
		}
		if attenuation > 1.0 {
			attenuation = 1.0
		}
		influence = attenuation
	} else {
		//influence = (d + 16) / (radius)
		d += 32
		d2 := math.Max(d-radius, 0)
		denominator := d2/radius + 1.0
		influence = 1.0 / (denominator * denominator)

		if influence > 1 {
			influence = 1
		}

		//if d+32 > radius {
		//	influence = 0
		//}
	}
	if x == 255 && y == 255 {
		glog.Infof("radius of light is %g; distance is %g; influence is %g", radius, d, influence)
	}

	//////
	// Additive light.
	// 16-bit R, G, B, A (though underlying precision is likely 8-bit).
	r, g, b, a := l.LightInfo.Color.RGBA()
	c.R += uint8(influence * float64(r>>8))
	c.G += uint8(influence * float64(g>>8))
	c.B += uint8(influence * float64(b>>8))
	//c.A += uint8(influence * float64(a>>8))
	c.A = uint8(255 * influence)
	a = a

	return c
}

func lightOverlayGen(width, height int, tileW, tileH int, light *Light, base image.Image) image.Image {
	fullSize := image.Rect(0, 0, width*tileW, height*tileH)
	overlay := &lightOverlay{bounds: fullSize, light: light, base: base}
	return overlay

	img := image.NewRGBA(fullSize)

	draw.Draw(img, fullSize, overlay, image.ZP, draw.Src)

	return img
}

type additiveOverlay struct {
	overlays []image.Image
	bounds   image.Rectangle
	base     image.Image

	ambientColor color.Color
	ambientLevel uint8

	// precomputed from ambientColor and ambientLevel:
	acR, acG, acB, acA uint32
}

func (*additiveOverlay) ColorModel() color.Model {
	return color.RGBAModel
}

func (o *additiveOverlay) Bounds() image.Rectangle {
	return o.bounds
}

func (o *additiveOverlay) precomputeAmbient() {
	// Ambient contains alpha-premultiplied RGBA.
	// 16-bit range integers stored in uint32.
	// o stands for output.
	//
	// We'll precompute this because uint<->float conversions might be
	// tedious on some platforms.
	or, og, ob, oa := o.ambientColor.RGBA()

	brightness := float64(o.ambientLevel) / 255.0
	oaF := float64(oa) * brightness

	oa = uint32(oaF)

	oaF /= 65535
	or = uint32(float64(or) * brightness * oaF)
	og = uint32(float64(og) * brightness * oaF)
	ob = uint32(float64(ob) * brightness * oaF)

	o.acR, o.acG, o.acB, o.acA = or, og, ob, oa
}

func (o additiveOverlay) At(x, y int) color.Color {
	if _, _, _, a := o.base.At(x, y).RGBA(); a == 0 {
		return color.RGBA{0, 0, 0, 0}
	}

	// Rather than computing every time, expect that these are precomputed.
	// o stands for output.
	//
	// 16-bit-range integers stored in uint32. (65535 is the brightest
	// value, despite the extra range.)
	or, og, ob, oa := o.acR, o.acG, o.acB, o.acA
	for _, overlay := range o.overlays {
		if oa > 65535 {
			oa = 65535
		}
		if oa > 1<<30 {
			oa = 0
		}

		r, g, b, a := overlay.At(x, y).RGBA()
		if false {
			// max of each component
			if r > or {
				or = r
			}
			if g > og {
				og = g
			}
			if b > ob {
				ob = b
			}
			if a > oa {
				oa = a
			}
		} else {
			if true {
				// additive
				or += r
				og += g
				ob += b
				oa += a
			} else {
				influence := float64(a) / 65535
				or += r - uint32(float64(r)*influence)
				og += g - uint32(float64(g)*influence)
				ob += b - uint32(float64(b)*influence)
				oa += a - uint32(float64(a)*influence)
			}
		}
	}
	if or > 65535 {
		or = 65535
	}
	if or > 1<<30 {
		or = 0
	}

	if og > 65535 {
		og = 65535
	}
	if og > 1<<30 {
		og = 0
	}

	if ob > 65535 {
		ob = 65535
	}
	if ob > 1<<30 {
		ob = 0
	}

	if oa > 65535 {
		oa = 65535
	}
	if oa > 1<<30 {
		oa = 0
	}

	// Just testing multiply...
	if true {
		br, bg, bb, ba := o.base.At(x, y).RGBA()
		or = uint32(((float64(br) / 65535) * (float64(or) / 65535)) * 65535)
		og = uint32(((float64(bg) / 65535) * (float64(og) / 65535)) * 65535)
		ob = uint32(((float64(bb) / 65535) * (float64(ob) / 65535)) * 65535)
		//oa = uint32(((float64(ba) / 65535) * (float64(oa) / 65535)) * 65535)
		oa = ba
	}
	// Just testing composite with srcalpha...
	if false {
		influence := float64(oa) / 65535
		br, bg, bb, ba := o.base.At(x, y).RGBA()
		or = uint32(((float64(br) / 65535) + influence*(float64(or)/65535)) * 65535)
		og = uint32(((float64(bg) / 65535) + influence*(float64(og)/65535)) * 65535)
		ob = uint32(((float64(bb) / 65535) + influence*(float64(ob)/65535)) * 65535)
		//oa = uint32(((float64(ba) / 65535) * (float64(oa) / 65535)) * 65535)
		oa = ba
	}

	return color.RGBA{
		R: uint8(or >> 8),
		G: uint8(og >> 8),
		B: uint8(ob >> 8),
		A: uint8(oa >> 8),
	}
}

func additiveOverlayGen(ambientColor color.Color, ambientLevel uint8, width, height int, tileW, tileH int, overlays []image.Image, base image.Image) image.Image {
	fullSize := image.Rect(0, 0, width*tileW, height*tileH)
	overlay := &additiveOverlay{bounds: fullSize, overlays: overlays, base: base, ambientColor: ambientColor, ambientLevel: ambientLevel}
	overlay.precomputeAmbient()
	return overlay

	// dead code:
	img := image.NewRGBA(fullSize)

	draw.Draw(img, fullSize, overlay, image.ZP, draw.Src)

	return img
}
