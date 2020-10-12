package things

import (
	"image"
	"image/color"
	"image/draw"

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

var (
	outfitColorLookupTable = [133]OutfitColor{
		0xFFFFFF, 0xFFD4BF, 0xFFE9BF, 0xFFFFBF, 0xE9FFBF, 0xD4FFBF,
		0xBFFFBF, 0xBFFFD4, 0xBFFFE9, 0xBFFFFF, 0xBFE9FF, 0xBFD4FF,
		0xBFBFFF, 0xD4BFFF, 0xE9BFFF, 0xFFBFFF, 0xFFBFE9, 0xFFBFD4,
		0xFFBFBF, 0xDADADA, 0xBF9F8F, 0xBFAF8F, 0xBFBF8F, 0xAFBF8F,
		0x9FBF8F, 0x8FBF8F, 0x8FBF9F, 0x8FBFAF, 0x8FBFBF, 0x8FAFBF,
		0x8F9FBF, 0x8F8FBF, 0x9F8FBF, 0xAF8FBF, 0xBF8FBF, 0xBF8FAF,
		0xBF8F9F, 0xBF8F8F, 0xB6B6B6, 0xBF7F5F, 0xBFAF8F, 0xBFBF5F,
		0x9FBF5F, 0x7FBF5F, 0x5FBF5F, 0x5FBF7F, 0x5FBF9F, 0x5FBFBF,
		0x5F9FBF, 0x5F7FBF, 0x5F5FBF, 0x7F5FBF, 0x9F5FBF, 0xBF5FBF,
		0xBF5F9F, 0xBF5F7F, 0xBF5F5F, 0x919191, 0xBF6A3F, 0xBF943F,
		0xBFBF3F, 0x94BF3F, 0x6ABF3F, 0x3FBF3F, 0x3FBF6A, 0x3FBF94,
		0x3FBFBF, 0x3F94BF, 0x3F6ABF, 0x3F3FBF, 0x6A3FBF, 0x943FBF,
		0xBF3FBF, 0xBF3F94, 0xBF3F6A, 0xBF3F3F, 0x6D6D6D, 0xFF5500,
		0xFFAA00, 0xFFFF00, 0xAAFF00, 0x54FF00, 0x00FF00, 0x00FF54,
		0x00FFAA, 0x00FFFF, 0x00A9FF, 0x0055FF, 0x0000FF, 0x5500FF,
		0xA900FF, 0xFE00FF, 0xFF00AA, 0xFF0055, 0xFF0000, 0x484848,
		0xBF3F00, 0xBF7F00, 0xBFBF00, 0x7FBF00, 0x3FBF00, 0x00BF00,
		0x00BF3F, 0x00BF7F, 0x00BFBF, 0x007FBF, 0x003FBF, 0x0000BF,
		0x3F00BF, 0x7F00BF, 0xBF00BF, 0xBF007F, 0xBF003F, 0xBF0000,
		0x242424, 0x7F2A00, 0x7F5500, 0x7F7F00, 0x557F00, 0x2A7F00,
		0x007F00, 0x007F2A, 0x007F55, 0x007F7F, 0x00547F, 0x002A7F,
		0x00007F, 0x2A007F, 0x54007F, 0x7F007F, 0x7F0055, 0x7F002A,
		0x7F0000,
	}
)

type OutfitColor int

func (col OutfitColor) RGBA() (r, g, b, a uint32) {
	r = uint32(outfitColorLookupTable[col]) >> 8 & 0xFF00
	g = uint32(outfitColorLookupTable[col]) & 0xFF00
	b = uint32(outfitColorLookupTable[col]) & 0xFF << 8
	a = 1
	return
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

func (c *Creature) ColorizedCreatureFrame(idx, dir int, outfitOverlayMask OutfitOverlayMask, colors []color.Color) image.Image {
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
func (c *Creature) CreatureFrame(idx, dir int, outfitOverlayMask OutfitOverlayMask, colorTemplate bool) image.Image {
	crf := creatureFrame{Frame: idx, Dir: dir, OutfitOverlayMask: outfitOverlayMask, ColorTemplate: colorTemplate}

	if c.img == nil {
		c.img = make(map[creatureFrame]image.Image)
	}

	if img, ok := c.img[crf]; ok {
		return img
	}

	// n.b. rendersize is used for scaling.
	gfx := c.outfit.GetGraphics()

	outfitOverlays := make([]int, 0, gfx.YDiv)
	for i := 0; i < int(gfx.YDiv); i++ {
		if i == 0 || outfitOverlayMask&1 != 0 && i < int(c.outfit.GetGraphics().YDiv) {
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
		innerImg := compositeGfx(idx, dir, y, 0, gfx, c.parent.spriteSet, []int{blendIdx})
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
