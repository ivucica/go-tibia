package gameworld

import (
	"image"
	"image/color"
	"image/draw"
	"math"

	"badc0de.net/pkg/go-tibia/dat"
	"badc0de.net/pkg/go-tibia/things"
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

func (o *compositeLightOverlay) At(x, y int) color.Color {
	if _, _, _, a := o.base.At(x, y).RGBA(); a == 0 {
		return color.RGBA{0, 0, 0, 0}
	}

	//var c color.Color
	//c = nightAmbient

	aR, aG, aB, aA := nightAmbient.RGBA()

	brightness := float64(nightAmbientLevel) / 255.0
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

func (o *additiveOverlay) At(x, y int) color.Color {
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

func CompositeMap(m MapDataSource, th *things.Things, x, y uint16, floorTop, floorBottom uint8, width, height int, tileW, tileH int) image.Image {
	fullSize := image.Rect(0, 0, width*tileW, height*tileH)
	img := image.NewRGBA(fullSize)

	var bottomRight image.Point

	ambientColor := nightAmbient
	ambientLevel := nightAmbientLevel

	// BUGS:
	//
	// Light is not quite calculated per floor.
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
