package things

import (
	"image"
	"image/color"
	"image/draw"
	"math"

	"badc0de.net/pkg/go-tibia/dat"
	"badc0de.net/pkg/go-tibia/otb/items"
	"badc0de.net/pkg/go-tibia/spr"
	"github.com/golang/glog"
)

func (i *Item) ItemFrame(idx int, x, y, z int) image.Image {
	itf := itemFrame{X: x, Y: y, Z: z, Frame: idx}

	if i.img == nil {
		i.img = make(map[itemFrame]image.Image)
	}

	if img, ok := i.img[itf]; ok {
		return img
	}

	if i.dataset == nil {
		if i.otb == nil {
			glog.Errorf("cannot composite image for item with unknown serverid (no i.otb set): no dat")
			return nil
		}
		glog.Errorf("cannot composite image for item %d: no dat", i.otb.Attributes[itemsotb.ITEM_ATTR_SERVERID].(uint16))
		return nil
	}

	// n.b. rendersize is used for scaling.
	gfx := i.dataset.GetGraphics()

	glog.V(2).Infof("compositing image for %d (client: %d): gfx: %+v", i.otb.Attributes[itemsotb.ITEM_ATTR_SERVERID].(uint16), i.dataset.Id, gfx)
	img := compositeGfx(idx, x, y, z, gfx, i.parent.spriteSet, nil)
	i.img[itf] = img
	return img
}

type OutfitOverlayMask int

const (
	OutfitOverlayMaskNone = OutfitOverlayMask(1 << iota)
	OutfitOverlayMaskFirst
	OutfitOverlayMaskSecond
	OutfitOverlayMaskThird
	OutfitOverlayMaskFourth

	OutfitOverlayMaskLast
)

type CreatureDirection int

const (
	CreatureDirectionNorth = CreatureDirection(0)
	CreatureDirectionEast  = CreatureDirection(iota)
	CreatureDirectionSouth
	CreatureDirectionWest
)

var (
	// grayLevels maps v (0-6) to a grayscale value.
	grayLevels = [7]uint8{
		0: 0xFF, 1: 0xDA, 2: 0xB6, 3: 0x91, 4: 0x6D, 5: 0x48, 6: 0x24,
	}

	// hueCodes maps h (1-18) to a 6-bit (R,G,B) 2-bit-per-component code.
	hueCodes = [19]uint8{
		0: 0, // Unused
		1: 0b110100, 2: 0b111000, 3: 0b111100,
		4: 0b101100, 5: 0b011100, 6: 0b001100,
		7: 0b001101, 8: 0b001110, 9: 0b001111,
		10: 0b001011, 11: 0b000111, 12: 0b000011,
		13: 0b010011, 14: 0b100011, 15: 0b110011,
		16: 0b110010, 17: 0b110001, 18: 0b110000,
	}
)

// valueLevels maps [v][c] to the final 8-bit component value.
// v = brightness (0-6), c = 2-bit code (0-3)
func valueLevels(v int, c uint8) uint8 {
    var val float64
    c_float := float64(c)

    switch v {
    case 0: // v=0 (Pastel/Tint)
        // Blends from 191 (0xBF) to 255 (0xFF)
        val = 191.0 + (64.0 * c_float / 3.0)
    case 1: // v=1 (Tone)
        // Blends from 143 (0x8F) to 191 (0xBF)
        val = 143.0 + (48.0 * c_float / 3.0)
    case 2: // v=2 (Tone)
        // Blends from 95 (0x5F) to 191 (0xBF)
        val = 95.0 + (96.0 * c_float / 3.0)
    case 3: // v=3 (Tone)
        // Blends from 63 (0x3F) to 191 (0xBF)
        val = 63.0 + (128.0 * c_float / 3.0)
    case 4: // v=4 (Pure)
        // Blends from 0 to 255 (0xFF)
        val = 255.0 * c_float / 3.0
    case 5: // v=5 (Shade)
        // Blends from 0 to 191 (0xBF)
        val = 191.0 * c_float / 3.0
    case 6: // v=6 (Dark Shade)
        // Blends from 0 to 127 (0x7F)
        val = 127.5 * c_float / 3.0
    }
    return uint8(math.Round(val))
}

type OutfitColor int

func (index OutfitColor) RGBA() (r, g, b, a uint32) {
	if index < 0 || index > 132 {
		return uint32(0), uint32(0), uint32(0), uint32(0xFF00)
	}

	// 1. Decompose the index
	v := int(index) / 19 // Brightness/Value (0-6)
	h := int(index) % 19 // Hue/Chroma (0-18)

	// 2. Handle Grayscale Case (h=0)
	if h == 0 {
		gray := uint32(grayLevels[v]) << 8
		return gray, gray, gray, uint32(0xFF00)
	}

	// 3. Handle Color Case (h > 0)
	// Get the 6-bit hue code (e.g., 0b110100)
	code := hueCodes[h]

	// Extract the 2-bit codes for R, G, and B
	rCode := (code >> 4) & 0b11
	gCode := (code >> 2) & 0b11
	bCode := (code >> 0) & 0b11

	// Look up the final 8-bit values from the valueLevels table
	r = uint32(valueLevels(v, rCode)) << 8
	g = uint32(valueLevels(v, gCode)) << 8
	b = uint32(valueLevels(v, bCode)) << 8

	return
}

func OutfitColorCount() int {
	return 133
}

func colorize(base image.Image, col color.Color, x, y int) color.RGBA {
	bpx := color.RGBAModel.Convert(base.At(x, y)).(color.RGBA)
	cpx := color.RGBAModel.Convert(col).(color.RGBA)

	const maxU32 = uint32(uint64(1)<<32 - 1)

	rF := float32(bpx.R) / float32(255)
	gF := float32(bpx.G) / float32(255)
	bF := float32(bpx.B) / float32(255)

	px := color.RGBA{
		R: byte(rF * float32(cpx.R)),
		G: byte(gF * float32(cpx.G)),
		B: byte(bF * float32(cpx.B)),
		A: byte(255),
	}
	return px
}

func (c *Creature) ColorizedCreatureFrame(idx int, dir CreatureDirection, outfitOverlayMask OutfitOverlayMask, colors []color.Color) image.Image {
	base := c.CreatureFrame(idx, dir, outfitOverlayMask, false)
	if c.outfit.GetGraphics().BlendFrames == 1 {
		return base
	}
	tpl := c.CreatureFrame(idx, dir, outfitOverlayMask, true)

	out := image.NewRGBA(base.Bounds())
	for y := 0; y < base.Bounds().Max.Y; y++ {
		for x := 0; x < base.Bounds().Max.X; x++ {
			tpx := tpl.At(x, y)

			rgba := color.RGBAModel.Convert(tpx).(color.RGBA)
			switch rgba {
			case color.RGBA{255, 0, 0, 255}:
				px := colorize(base, colors[0], x, y)
				out.Set(x, y, px)
			case color.RGBA{0, 255, 0, 255}:
				px := colorize(base, colors[1], x, y)
				out.Set(x, y, px)
			case color.RGBA{0, 0, 255, 255}:
				px := colorize(base, colors[2], x, y)
				out.Set(x, y, px)
			case color.RGBA{255, 255, 0, 255}:
				px := colorize(base, colors[3], x, y)
				out.Set(x, y, px)
			default:
				out.Set(x, y, base.At(x, y))
			}

		}
	}

	return out
}
func (c *Creature) CreatureFrame(idx int, dir CreatureDirection, outfitOverlayMask OutfitOverlayMask, colorTemplate bool) image.Image {
	crf := creatureFrame{Frame: idx, Dir: dir, OutfitOverlayMask: outfitOverlayMask, ColorTemplate: colorTemplate}

	if c.img == nil {
		c.img = make(map[creatureFrame]image.Image)
	}

	if img, ok := c.img[crf]; ok {
		return img
	}

	// n.b. rendersize is used for scaling.
	gfx := c.outfit.GetGraphics()

	glog.Infof("overlay mask %01x, ydiv %d", outfitOverlayMask, gfx.YDiv)
	outfitOverlays := make([]int, 0, gfx.YDiv)
	for i := 0; i < int(gfx.YDiv); i++ {
		if i == 0 || (outfitOverlayMask&1 != 0 && i < int(gfx.YDiv)) {
			glog.Infof(" -> add overlay mask %d", i)
			outfitOverlays = append(outfitOverlays, i)
		}
		outfitOverlayMask >>= 1
	}

	blendIdx := 0
	if colorTemplate {
		blendIdx = 1
	}

	var img image.Image
	for _, y := range outfitOverlays {
		innerImg := compositeGfx(idx, int(dir), y, 0, gfx, c.parent.spriteSet, []int{blendIdx})
		if img == nil {
			img = innerImg
		} else {
			draw.Draw(img.(draw.Image), img.Bounds(), innerImg, image.ZP, draw.Over)
		}
	}

	c.img[crf] = img
	return img
}

func compositeGfx(idx int, x, y, z int, gfx *dat.Graphics, s *spr.SpriteSet, blendFrames []int) image.Image {
	w := int(gfx.RenderSize)
	h := int(gfx.RenderSize)
	if w == 0 || h == 0 {
		w = 32 * int(gfx.Width)
		h = 32 * int(gfx.Height)
	}

	img := image.NewRGBA(image.Rect(0, 0, int(gfx.RenderSize), int(gfx.RenderSize)))

	x %= int(gfx.XDiv)
	y %= int(gfx.YDiv)
	z %= int(gfx.ZDiv)
	idx %= int(gfx.AnimCount)

	spriteSize := int(gfx.BlendFrames) * int(gfx.Height) * int(gfx.Width)

	// TODO: z
	activeSprite := spriteSize * (x + y*int(gfx.XDiv) + idx*int(gfx.XDiv)*int(gfx.YDiv))

	if len(blendFrames) == 0 {
		blendFrames = make([]int, 0, gfx.BlendFrames)
		for i := 0; i < int(gfx.BlendFrames); i++ {
			blendFrames = append(blendFrames, i)
		}
	}

	for _, b := range blendFrames {
		now := activeSprite + int(gfx.Width)*int(gfx.Height)*b
		if now >= len(gfx.Sprites) {
			continue
		}

		for y := 0; y < int(gfx.Height); y++ {
			for x := 0; x < int(gfx.Width); x++ {
				spr := gfx.Sprites[now]
				now++
				if spr == 0 {
					continue
				}
				src := s.Image(int(spr))
				if src == nil {
					glog.Errorf("error decoding sprite %d", spr)
					continue
				}
				r := image.Rect(
					w-(x)*32, h-(y)*32,
					w-(x+1)*32, h-(y+1)*32)

				draw.Draw(img, r, src, image.ZP, draw.Over)
			}
		}
	}

	return img
}
