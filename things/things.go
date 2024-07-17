package things

import (
	"badc0de.net/pkg/go-tibia/dat"
	"badc0de.net/pkg/go-tibia/otb/items"
	"badc0de.net/pkg/go-tibia/spr"

	"github.com/golang/glog"

	"fmt"
	"image"
	"strings"
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

func (i *Item) Article() string {
	return i.otb.Article()
}

func (i *Item) Description() string {
	return i.otb.Description()
}

func (i *Item) Temp__ExternalLink() string {
	if i.Name() == "" || i.Name() == "unnamed item" {
		return ""
	}
	return "https://tibia.fandom.com/wiki/" + strings.Replace(strings.Replace(strings.Title(i.Name()), " ", "_", -1), " Of ", " of ", -1)
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

func (i *Item) ClientID(clientVersion uint16) uint16 {
	return i.otb.ClientID()
}

func (i *Item) ServerID() uint16 {
	return i.otb.ServerID()
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

// Item returns the things-representation of an item which has the passed server
// (otb) ID. The used dataset will be for the passed version, if the item even
// exists for the passed version.
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

// ItemWithSequentialOTBID returns Nth item in OTB storage. This is
// useful only for pagination purposes, and the index may change depending on
// the client version.
//
// idx is zero-based.
func (t *Things) ItemWithSequentialOTBIDX(idx int, clientVersion uint16) (*Item, error) {
	if int(idx) < 0 || int(idx) >= len(t.items.Items) {
		return nil, fmt.Errorf("item at index %d in v%d not present in otb (sz: %d)", idx, clientVersion, len(t.items.Items))
	}

	otb := t.items.Items[idx]
	serverID := otb.ServerID()

	//serverID := t.items.ExtantServerItemIDs[idx]
	glog.Infof("item at idx %d has server id %d", idx, serverID)
	return t.Item(serverID, clientVersion) // FIXME: this exists BECAUSE we don't want a lookup by server id! avoid invoking t.Item which uses server id
}

// ItemWithSequentialClientID returns Nth item with valid client ID. This is
// useful only for pagination purposes, and the index may change depending on
// the client version.
//
// idx is zero-based.
func (t *Things) ItemWithSequentialClientID(idx uint16, clientVersion uint16) (*Item, error) {
	if int(idx) < 0 || int(idx) >= len(t.items.ExtantClientItemArrayIdxs) {
		return nil, fmt.Errorf("item at index %d in v%d not present among otb known client IDs (sz: %d)", idx, clientVersion, len(t.items.ExtantClientItemArrayIdxs))
	}

	off := t.items.ExtantClientItemArrayIdxs[idx]
	return t.ItemWithSequentialOTBIDX(off, clientVersion)
}

// ItemWithSequentialServerID returns Nth item with valid server ID. This is
// useful only for pagination purposes, and the index may change depending on
// the server version.
//
// idx is zero-based.
func (t *Things) ItemWithSequentialServerID(idx uint16, clientVersion uint16) (*Item, error) {
	if int(idx) < 0 || int(idx) >= len(t.items.ExtantServerItemArrayIdxs) {
		return nil, fmt.Errorf("item at index %d in v%d not present among otb known server IDs (sz: %d)", idx, clientVersion, len(t.items.ExtantServerItemArrayIdxs))
	}

	off := t.items.ExtantServerItemArrayIdxs[idx]
	return t.ItemWithSequentialOTBIDX(off, clientVersion)
}

// ItemWithClientID returns the things-representation of an item which has the
// passed client ID in the passed client version. The used dataset will be for
// the passed version.
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

// MinItemClientID returns the minimum ID of an item for the passed version.
func (t *Things) MinItemClientID(clientVersion uint16) uint16 {
	if t.dataset != nil {
		return t.dataset.MinItemID()
	}
	return t.items.MinClientID
}

// MaxItemClientID returns the largest ID of an item for the passed version.
//
// This can be sourced either from OTB or from dataset, with preference for dataset.
func (t *Things) MaxItemClientID(clientVersion uint16) uint16 {
	if t.dataset != nil {
		return t.dataset.MaxItemID()
	}
	return t.items.MaxClientID
}

// MinItemServerID returns the minimum ID of an item for the passed version.
func (t *Things) MinItemServerID(clientVersion uint16) uint16 {
	return t.items.MinServerID
}

// MaxItemServerID returns the maximum ID of an item for the passed version.
//
// Due to gaps, not the same as ItemCount.
func (t *Things) MaxItemServerID(clientVersion uint16) uint16 {
	return t.items.MaxServerID
}

// ItemCount returns the serverside items count that are known for the passed
// version.
func (t *Things) ItemCount(clientVersion uint16) int {
	return len(t.items.Items)
}

// ClientItemCount returns the serverside items count that have a known client
// ID for the passed version.
func (t *Things) ClientItemCount(clientVersion uint16) int {
	return len(t.items.ExtantClientItemArrayIdxs)
}

// ServerItemCount returns the serverside items count that have a known server
// ID for the passed version.
func (t *Things) ServerItemCount(clientVersion uint16) int {
	return len(t.items.ExtantServerItemArrayIdxs)
}

// Creature returns the things-representation of a creature / outfit which has
// the passed server ID. The used dataset will be for the passed version, if the
// creature even exists for the passed version.
//
// Currently there is no distinction between client and server IDs. Yet.
func (t *Things) Creature(serverID uint16, clientVersion uint16) (*Creature, error) {
	// Currently there is no distinction between client and server IDs. Yet.
	return t.CreatureWithClientID(serverID, clientVersion)
}

// CreatureWithClientID returns the things-representation of a creature which
// has the passed client ID in the passed client version. The outfit returned
// will have the dataset description for the passed version.
func (t *Things) CreatureWithClientID(clientID uint16, clientVersion uint16) (*Creature, error) {
	return &Creature{
		clientID: clientID,
		outfit:   t.dataset.Outfit(clientID),
		parent:   t,
	}, nil
}

// CreatureCount returns the serverside creatures / outfits count that are known
// for the passed version.
//
// Currently there is no distinction between client and server IDs, so the
// dataset count is returned.
func (t *Things) CreatureCount(clientVersion uint16) int {
	return t.dataset.OutfitCount()
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

func (t *Things) Temp__GetServerItemArrayOffsetInOTB(serverID uint16, clientVersion uint16) int {
	// Useful when we have to paginate by all server items we know, due to gaps.
	//
	// Its lets us do an ugly hack of offsetting inside otb-array (which is how
	// gotweb displays serverside items) to display just one item based on its
	// sid. If that hack is gone, this function should be gone too.
	if t == nil {
		glog.Errorf("Things.Temp__GetItemArrayOffsetInOTB: things is null")
		return -1
	}
	if t.items == nil {
		glog.Errorf("Things.Temp__GetItemArrayOffsetInOTB: items is null")
		return -1
	}
	if idx, ok := t.items.ServerIDToArrayIndex[serverID]; ok {
		return idx
	}
	return -1
}

func (t *Things) Temp__GetKnownClientIDItemArrayOffsetInOTB(serverID uint16, clientVersion uint16) int {
	// Similar pagination hack.
	if t == nil {
		glog.Errorf("Things.Temp__GetItemArrayOffsetInOTB: things is null")
		return -1
	}
	if t.items == nil {
		glog.Errorf("Things.Temp__GetItemArrayOffsetInOTB: items is null")
		return -1
	}
	if idx, ok := t.items.ServerIDToExtantClientItemArrayIDXs[serverID]; ok {
		return idx
	}
	return -1
}

func (t *Things) Temp__DATItemCount(clientVersion uint16) int {
	return t.dataset.ItemCount()
}
