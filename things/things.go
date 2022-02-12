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

func (i *Item) MapColor() dat.DatasetColor {
	// TODO: return MapColorOK. currently omitted to allow use in template
	return i.dataset.MapColor //, i.dataset.MapColorOK
}

func (i *Item) MapColorOK() bool {
	// TODO: maybe call HasMapColor
	return i.dataset.MapColorOK
}

func (i *Item) LightInfo() dat.LightInfo {
	return i.dataset.LightInfo
}

func (i *Item) GraphicsSize() struct{ W, H int } {
	gfx := i.dataset.GetGraphics()
	// FIXME: multiplying .Width and .Height for client item 6469 (wooden window) and others by gfx.RenderSize gave us 128x128
	// for 6473 (planks) we got 54x128 (which is interestingly 54x54)
	//
	// perhaps it is best to just use RenderSize
	return struct{ W, H int }{W: int(gfx.RenderSize), H: int(gfx.RenderSize)}
}

func (i *Item) ValidClientItem() bool {
	return i.dataset != nil
}

// RawClientDatasetItem780 is for debug or viewing use only; please do not
// access it outside these scenarios, as it may disappear anytime.
//
// It is currently used for showing opt byte descriptions in the web UI.
//
// It may return nil.
func (i *Item) RawClientDatasetItem780() *dat.Item {
	return i.dataset
}

type Creature struct {
	clientID uint16
	outfit   *dat.Outfit

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
	return struct{ W, H int }{W: int(gfx.RenderSize), H: int(gfx.RenderSize)}
	}

func (c *Creature) IdleAnim() bool {
	return c.outfit.IdleAnim
}

func (c *Creature) AnimCount() int {
	gfx := c.outfit.GetGraphics()
	return int(gfx.AnimCount)
}

func (c *Creature) ClientID(clientVersion uint16) uint16 {
	return c.clientID
}

func (c *Creature) ServerID() int {
	return int(c.clientID)
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

	i := &Item{
		otb:    otb,
		parent: t,
	}
	if datID != 0 {
		i.dataset = t.dataset.Item(datID)
	}
	return i, nil
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

func (t *Things) Creature(serverID uint16, clientVersion uint16) (*Creature, error) {
	// Currently there is no distinction between client and server IDs. Yet.
	return t.CreatureWithClientID(serverID, clientVersion)
}

func (t *Things) CreatureWithClientID(clientID uint16, clientVersion uint16) (*Creature, error) {
	return &Creature{
		clientID: clientID,
		outfit:   t.dataset.Outfit(clientID),
		parent:   t,
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
