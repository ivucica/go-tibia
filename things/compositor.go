package things

import (
	"image"
	"image/draw"

	"badc0de.net/pkg/go-tibia/otb/items"
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
	
	// n.b. rendersize is used for scaling.
	gfx := i.dataset.GetGraphics()
	img := image.NewRGBA(image.Rect(0, 0, int(gfx.Width)*int(gfx.RenderSize), int(gfx.Height)*int(gfx.RenderSize)))
	glog.V(2).Infof("compositing image for %d (client: %d): %+v - gfx: %+v", i.otb.Attributes[itemsotb.ITEM_ATTR_SERVERID].(uint16), i.dataset.Id, img, gfx)

	x %= int(gfx.XDiv)
	y %= int(gfx.YDiv)
	z %= int(gfx.ZDiv)
	idx %= int(gfx.AnimCount)

	spriteSize := int(gfx.BlendFrames) * int(gfx.Height) * int(gfx.Width)

	// TODO: z
	activeSprite := spriteSize * (x + y*int(gfx.XDiv) + idx*int(gfx.XDiv)*int(gfx.YDiv))

	var now int
	now = activeSprite

	for b := 0; b < int(gfx.BlendFrames); b++ {
		for y := 0; y < int(gfx.Height); y++ {
			for x := 0; x < int(gfx.Width); x++ {
				spr := gfx.Sprites[now]
				now++
				if spr == 0 {
					continue
				}
				src := i.parent.spriteSet.Image(int(spr))
				if src == nil {
					glog.Errorf("error decoding sprite %d", spr)
					continue
				}
				r := image.Rect(
					(int(gfx.Width)-x-1)*int(gfx.RenderSize), (int(gfx.Height)-y-1)*int(gfx.RenderSize),
					(int(gfx.Width)-x-1+1)*int(gfx.RenderSize), (int(gfx.Height)-y-1+1)*int(gfx.RenderSize))
				draw.Draw(img, r, src, image.ZP, draw.Over)
			}
		}
	}

	i.img[itf] = img
	return img
}
