package things

import (
	"badc0de.net/pkg/go-tibia/dat"
	"badc0de.net/pkg/go-tibia/otb/items"
	"badc0de.net/pkg/go-tibia/spr"

	"github.com/golang/glog"

	"image"
)

type Item struct {
	otb     *itemsotb.Item
	dataset *dat.Item

	parent *Things

	img map[itemFrame]image.Image
}
type itemFrame struct{ X, Y, Z, Frame int }

func (i *Item) Name() string {
	return i.otb.Name()
}

func (i *Item) LightInfo() dat.LightInfo {
	return i.dataset.LightInfo
}

func (i *Item) GraphicsSize() struct{ W, H int } {
	gfx := i.dataset.GetGraphics()
	return struct{ W, H int }{W: int(gfx.Width * gfx.RenderSize), H: int(gfx.Height * gfx.RenderSize)}
}

type Creature struct {
	outfit *dat.Outfit

	parent *Things

	img map[creatureFrame]image.Image
}
type creatureFrame struct {
	Dir               CreatureDirection
	Frame             int
	OutfitOverlayMask OutfitOverlayMask
	ColorTemplate     bool
}

func (c *Creature) Name() string {
	return "a creature"
}

func (c *Creature) LightInfo() dat.LightInfo {
	return c.outfit.LightInfo
}

func (c *Creature) GraphicsSize() struct{ W, H int } {
	gfx := c.outfit.GetGraphics()
	return struct{ W, H int }{W: int(gfx.Width * gfx.RenderSize), H: int(gfx.Height * gfx.RenderSize)}
}

func (c *Creature) IdleAnim() bool {
	return c.outfit.IdleAnim
}

func (c *Creature) AnimCount() int {
	gfx := c.outfit.GetGraphics()
	return int(gfx.AnimCount)
}

type Things struct {
	items     *itemsotb.Items
	dataset   *dat.Dataset
	spriteSet *spr.SpriteSet
}

func New() (*Things, error) {
	return &Things{}, nil
}

func (t *Things) AddItemsOTB(i *itemsotb.Items) error {
	t.items = i
	return nil
}

func (t *Things) AddTibiaDataset(d *dat.Dataset) error {
	t.dataset = d
	return nil
}

func (t *Things) AddSpriteSet(s *spr.SpriteSet) error {
	t.spriteSet = s
	return nil
}

func (t *Things) TibiaDatasetSignature() uint32 {
	return t.dataset.Header.Signature
}

func (t *Things) SpriteSetSignature() uint32 {
	return t.spriteSet.Header.Signature
}

func (t *Things) Item(serverID uint16, clientVersion uint16) (*Item, error) {
	otb := t.Temp__GetItemFromOTB(serverID, clientVersion)
	datID := t.Temp__GetClientIDForServerID(serverID, clientVersion)
	return &Item{
		otb:     otb,
		dataset: t.dataset.Item(datID),
		parent:  t,
	}, nil
}

func (t *Things) ItemWithClientID(clientID uint16, clientVersion uint16) (*Item, error) {
	itm, err := t.items.ItemByClientID(clientID)
	if err != nil {
		glog.Errorf("item %d fetch gave error: %v", clientID, err)
		return nil, err
	}
	return &Item{
		otb:     itm,
		dataset: t.dataset.Item(clientID),
		parent:  t,
	}, nil
}

func (t *Things) CreatureWithClientID(clientID uint16, clientVersion uint16) (*Creature, error) {
	return &Creature{
		outfit: t.dataset.Outfit(clientID),
		parent: t,
	}, nil
}

func (t *Things) Temp__GetClientIDForServerID(serverID uint16, clientVersion uint16) uint16 {
	itm, err := t.items.ItemByServerID(serverID)
	if err != nil {
		glog.Errorf("item %d fetch gave error: %v", serverID, err)
		return 0
	}
	if attr, ok := itm.Attributes[itemsotb.ITEM_ATTR_CLIENTID]; ok {
		return attr.(uint16)
	} else {
		glog.Errorf("item %d has no ITEM_ATTR_CLIENTID", serverID)
		return 0
	}
}

func (t *Things) Temp__GetItemFromOTB(serverID uint16, clientVersion uint16) *itemsotb.Item {
	if t == nil {
		glog.Errorf("Things.Temp__GetItemFromOTB: things is null")
		return nil
	}
	if t.items == nil {
		glog.Errorf("Things.Temp__GetItemFromOTB: items is null")
		return nil
	}
	itm, err := t.items.ItemByServerID(serverID)
	if err != nil {
		return nil
	}
	return itm
}
