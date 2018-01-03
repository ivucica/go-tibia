package dat

import (
	"encoding/binary"
	"fmt"
	"io"

	"github.com/golang/glog"
)

// Enumeration of versions that are 'actively' supported.
//
// (Values may change between versions of the package and have no meaning.)
const (
	CLIENT_VERSION_UNKNOWN = iota
	CLIENT_VERSION_854
)

// Dataset represents a set of items, outfits, effects and distance
// effects read from a Tibia dataset file ('things' or 'dataset entries').
type Dataset struct {
	header header

	items           []Item
	outfits         []Outfit
	effects         []Effect
	distanceEffects []DistanceEffect
}

type header struct {
	Signature                                                uint32
	ItemCount, OutfitCount, EffectCount, DistanceEffectCount uint16 // TODO(ivucica): rename to 'max id'
}

type endOfOptBlock struct{ error }

func (endOfOptBlock) Error() string {
	return "quasi-error used to signal end of opt byte block"
}

// DatasetEntry interface represents an entry in the dataset.
//
// It exposes a method to get a description of graphics for a particular 'thing',
// but is primarily intended to make it easier to abstract representing 'any entry'
// in the type system.
//
// All dataset entries have graphics attached to them.
type DatasetEntry interface {
	GetGraphics() *Graphics
}

// Item represents a dataset entry describing an in-game item.
//
// Items range from ground tiles, through wall tiles, to inventory items such as
// apples or swords.
type Item struct {
	DatasetEntry
	Graphics

	Id int

	GroundSpeed uint16
	SortOrder   uint16
	Container   bool
	Stackable   bool
	AlwaysUsed  bool
	Usable      bool
	Rune        bool

	Readable bool
	Writable bool
	MaxRWLen uint16

	FluidContainer bool
	Splash         bool

	BlockingPlayer   bool
	Immobile         bool
	BlockingMissiles bool
	BlockingMonsters bool

	Equipable      bool
	Hangable       bool
	HorizontalItem bool
	VerticalItem   bool
	RotatableItem  bool

	LightInfo
	OffsetInfo
	PlayerOffset uint16
	LargeOffset  bool
	IdleAnim     bool
	MapColor     uint16
	LookThrough  bool
}
// Outfit represents a dataset entry describing a possible appearance for an in-game character, whether NPC or player.
type Outfit struct {
	DatasetEntry
	Graphics

	Id int
	OffsetInfo
	IdleAnim bool
	LightInfo
}
// Effect represents a temporarily-appearing in-game effect, such as a poof of smoke, then disappears.
type Effect struct {
	DatasetEntry
	Graphics

	Id int
	LightInfo
}
// DistanceEffect represents an in-game effect which moves from one tile to another over a period of time, then diappears.
type DistanceEffect struct {
	DatasetEntry
	Graphics

	Id int
	LightInfo
}

// LightInfo represents a recurring structure in the binary format of the data file concerning the strength and color of the light emitted client-side.
type LightInfo struct {
	Strength, Color uint16
}
// OffsetInfo represents how far the object should be drawn offset to its usual drawing location.
type OffsetInfo struct {
	X, Y uint16
}

// GraphicsDimensions represents size of an item expressed in tiles.
//
// It's extracted as a type so it can more easily be read from the binary file.
type GraphicsDimensions struct {
	// How many on-screen tiles will this sprite take, when drawn.
	Width, Height uint8
}

// GraphicsDetails represents various points detailing how many individual sprites compose
// a thing, and how should they be drawn.
//
// This includes number of animation frames, number of sprites that should be layered
// one on top of the other, etc.
type GraphicsDetails struct {
	// How many WxH blocks will be drawn one on top of the other.
	BlendFrames uint8
	// How many variations does this sprite have based on its position on the map.
	XDiv, YDiv, ZDiv uint8
	// How many blocks of WxHxBlendFramesxXdivxYdivxZdiv does this sprite use as animation frames.
	AnimCount uint8
}

// Graphics describes sprites associated with a particular thing, and how
// they should be drawn.
type Graphics struct {
	// WxH. Separated into a struct for easier reading.
	GraphicsDimensions
	// How many pixels should each sprite's tile take on screen? Usually 32.
	RenderSize uint8
	// Details on how to render each of the WxH sprite blocks (animations, variations, etc.)
	// Separated into a struct for easier reading.
	GraphicsDetails

	// Which sprites are used when rendering.
	Sprites []uint16
}

// String returns a string representation for an item.
func (i Item) String() string {
	return fmt.Sprintf("item %d", i.Id)
}

// IsGround returns information whether an item is a ground item.
//
// Currently this is based on existence of an item's ground speed.
func (i Item) IsGround() bool {
	return i.GroundSpeed != 0
}

// GetGraphics returns sprites associated with this item.
func (i Item) GetGraphics() *Graphics {
	return &i.Graphics
}

// String returns a string representation for an outfit.
func (o Outfit) String() string {
	return fmt.Sprintf("outfit %d", o.Id)
}

// GetGraphics returns sprites associated with this outfit.
func (o Outfit) GetGraphics() *Graphics {
	return &o.Graphics
}

// String returns a string representation for an effect.
func (e Effect) String() string {
	return fmt.Sprintf("effect %d", e.Id)
}

// GetGraphics returns sprites associated with this effect.
func (e Effect) GetGraphics() *Graphics {
	return &e.Graphics
}

// String returns a string representation for a distance effect.
func (d DistanceEffect) String() string {
	return fmt.Sprintf("distance effect %d", d.Id)
}

// GetGraphics returns sprites associated with this distance effect.
func (d DistanceEffect) GetGraphics() *Graphics {
	return &d.Graphics
}

// NewDataset reads the dataset file from the passed io.Reader and returns the Dataset object.
func NewDataset(r io.Reader) (*Dataset, error) {
	glog.V(3).Info("starting to read dataset")
	h := header{}
	if err := binary.Read(r, binary.LittleEndian, &h); err != nil {
		return nil, fmt.Errorf("error reading dataset header: %v", err)
	}

	glog.V(3).Infof("creating dataset")
	dataset := Dataset{
		header:          h,
		items:           make([]Item, h.ItemCount-100+1),
		outfits:         make([]Outfit, h.OutfitCount),
		effects:         make([]Effect, h.EffectCount),
		distanceEffects: make([]DistanceEffect, h.DistanceEffectCount),
	}

	glog.V(3).Infoln("loading...")
	err := dataset.load780plus(r)
	if err != nil {
		return nil, fmt.Errorf("error reading the dataset: %v", err)
	}
	glog.V(3).Infoln("loaded")

	return &dataset, nil
}

// load780plus loads the format used in game version 7.8 and later.
func (d *Dataset) load780plus(r io.Reader) error {
	var e DatasetEntry
	for id := 0; id < len(d.items)+len(d.outfits)+len(d.effects)+len(d.distanceEffects); id++ {
		e = nil
		eid := 0
		if id < len(d.items) {
			eid = id
			glog.V(4).Infof("loading item id %d", eid)
			e = &d.items[eid]
			e.(*Item).Id = eid + 100
		} else if id < len(d.items)+len(d.outfits) {
			eid = id - len(d.items)
			glog.V(4).Infof("loading outfit id %d", eid)
			e = &d.outfits[eid]
			e.(*Outfit).Id = eid + 1
		} else if id < len(d.items)+len(d.outfits)+len(d.effects) {
			eid = id - len(d.items) - len(d.outfits)
			glog.V(4).Infof("loading effect id %d", eid)
			e = &d.effects[eid]
			e.(*Effect).Id = eid + 1
		} else if id < len(d.items)+len(d.outfits)+len(d.effects)+len(d.distanceEffects) {
			eid = id - len(d.items) - len(d.outfits) - len(d.effects)
			glog.V(4).Infof("loading distance effect id %d", eid)
			e = &d.distanceEffects[eid]
			e.(*DistanceEffect).Id = eid + 1
		}
		glog.V(4).Infof("id %d (%d, %d, %d, %d) %T", id, len(d.items), len(d.items)+len(d.outfits), len(d.items)+len(d.outfits)+len(d.effects), len(d.items)+len(d.outfits)+len(d.effects)+len(d.distanceEffects), e)
		if e == nil {
			glog.Errorf("unsupported dataset entry type at index %d", id)
			return fmt.Errorf("unsupported dataset entry type at index %d", id)
		}
		glog.V(4).Infoln("load bytes for ", eid)
		if err := d.load780OptBytes(r, e); err != nil {
			return fmt.Errorf("error reading optbytes for %s: %v", e, err)
		}
		glog.V(4).Infoln("load spec")

		if err := d.loadGraphicsSpec(r, e); err != nil {
			return fmt.Errorf("error reading graphics spec for %s: %v", e, err)
		}
		glog.V(4).Infoln("next item")
	}
	glog.V(3).Infof("done with 780 dataset")
	return nil
}

// ClientVersion returns which version of the game this data file comes from.
//
// Currently supported is only 8.54.
func (d Dataset) ClientVersion() int {
	if d.header.Signature == 0x4b28b89e || d.header.Signature == 0x4b1e2caa {
		return CLIENT_VERSION_854
	}
	return CLIENT_VERSION_UNKNOWN
}

// load780OptBytes reads all option bytes from the passed reader, configuring the passed dataset entry.
//
// load780OptByte is repeatedly invoked.
func (d *Dataset) load780OptBytes(r io.Reader, e DatasetEntry) error {
	var prevOptByte uint8
	for {
		optByte, err := d.load780OptByte(r, e)
		if err, ok := err.(*endOfOptBlock); err != nil && ok {
			return nil
		}
		if err != nil {
			return fmt.Errorf("optbyte 0x%x: %v (previous optbyte: 0x%x)", optByte, err, prevOptByte)
		}
		prevOptByte = optByte
	}
}

// load780OptByte reads a single option byte from the passed reader and configures the passed dataset entry.
//
// Byte that was just read is returned.
func (d *Dataset) load780OptByte(r io.Reader, e DatasetEntry) (uint8, error) {
	var optByte uint8
	err := binary.Read(r, binary.LittleEndian, &optByte)
	if err != nil {
		return 0, fmt.Errorf("error reading the opt byte: %v", err)
	}

	switch optByte {
	case 0x00: // Ground tile.
		i, ok := e.(*Item)
		if !ok {
			return optByte, fmt.Errorf("non-item ground tile")
		}
		err := binary.Read(r, binary.LittleEndian, &i.GroundSpeed)
		if err != nil {
			return optByte, fmt.Errorf("error reading the ground tile speed: %v", err)
		}

	case 0x01: // On-top items.
		i, ok := e.(*Item)
		if !ok {
			return optByte, fmt.Errorf("non-item on-top entry")
		}
		i.SortOrder = 1

	case 0x02: // Walk-through items (e.g. doors).
		i, ok := e.(*Item)
		if !ok {
			return optByte, fmt.Errorf("non-item walk-through-1 entry")
		}
		i.SortOrder = 2

	case 0x03: // Higher walk-through items (e.g. arches).
		i, ok := e.(*Item)
		if !ok {
			return optByte, fmt.Errorf("non-item walk-through-2 entry")
		}
		i.SortOrder = 3

	case 0x04: // Container item.
		i, ok := e.(*Item)
		if !ok {
			return optByte, fmt.Errorf("non-item container entry")
		}
		i.Container = true

	case 0x05: // Stackable item.
		i, ok := e.(*Item)
		if !ok {
			return optByte, fmt.Errorf("non-item stackable entry")
		}
		i.Stackable = true

	case 0x06: // Always used. (e.g. ladders)
		i, ok := e.(*Item)
		if !ok {
			return optByte, fmt.Errorf("non-item always-used entry")
		}
		i.AlwaysUsed = true

	case 0x07: // Usable.
		i, ok := e.(*Item)
		if !ok {
			return optByte, fmt.Errorf("non-item usable entry")
		}
		i.Usable = true

	case 0x08: // Rune.
		i, ok := e.(*Item)
		if !ok {
			return optByte, fmt.Errorf("non-item usable entry")
		}
		i.Rune = true

	case 0x09: // R/W item.
		i, ok := e.(*Item)
		if !ok {
			return optByte, fmt.Errorf("non-item RW item entry")
		}
		i.Readable = true
		i.Writable = true
		err := binary.Read(r, binary.LittleEndian, &i.MaxRWLen)
		if err != nil {
			return optByte, fmt.Errorf("error reading RW item info: %v", err)
		}

	case 0x0A: // RO item.
		i, ok := e.(*Item)
		if !ok {
			return optByte, fmt.Errorf("non-item RO item entry")
		}
		i.Readable = true
		err := binary.Read(r, binary.LittleEndian, &i.MaxRWLen)
		if err != nil {
			return optByte, fmt.Errorf("error reading RO item info: %v", err)
		}

	case 0x0B: // Fluid container.
		i, ok := e.(*Item)
		if !ok {
			return optByte, fmt.Errorf("non-item fluid container")
		}
		i.FluidContainer = true

	case 0x0C: // Splash.
		i, ok := e.(*Item)
		if !ok {
			return optByte, fmt.Errorf("non-item splash")
		}
		i.Splash = true

	case 0x0D: // Blocking player.
		i, ok := e.(*Item)
		if !ok {
			return optByte, fmt.Errorf("non-item blocker")
		}
		i.BlockingPlayer = true

	case 0x0E: // Immobile item.
		i, ok := e.(*Item)
		if !ok {
			return optByte, fmt.Errorf("non-item immobile")
		}
		i.Immobile = true

	case 0x0F: // Blocking missiles.
		i, ok := e.(*Item)
		if !ok {
			return optByte, fmt.Errorf("non-item blocker for missiles")
		}
		i.BlockingMissiles = true

	case 0x10: // Blocking monsters.
		i, ok := e.(*Item)
		if !ok {
			return optByte, fmt.Errorf("non-item blocker for monsters")
		}
		i.BlockingMonsters = true

	case 0x11: // Equipable.
		i, ok := e.(*Item)
		if !ok {
			return optByte, fmt.Errorf("non-item equipable")
		}
		i.Equipable = true

	case 0x12: // Hangable.
		i, ok := e.(*Item)
		if !ok {
			return optByte, fmt.Errorf("non-item equipable")
		}
		i.Hangable = true

	case 0x13: // Horizontal item.
		i, ok := e.(*Item)
		if !ok {
			return optByte, fmt.Errorf("non-item horizontal")
		}
		i.HorizontalItem = true

	case 0x14: // Vertical item.
		i, ok := e.(*Item)
		if !ok {
			return optByte, fmt.Errorf("non-item vertical")
		}
		i.VerticalItem = true

	case 0x15: // Rotatable item.
		i, ok := e.(*Item)
		if !ok {
			return optByte, fmt.Errorf("non-item rotatable")
		}
		i.RotatableItem = true

	case 0x16: // Lightcaster.
		if i, ok := e.(*Item); ok {
			err := binary.Read(r, binary.LittleEndian, &i.LightInfo)
			if err != nil {
				return optByte, fmt.Errorf("error reading light info: %v", err)
			}
		} else if o, ok := e.(*Outfit); ok {
			err := binary.Read(r, binary.LittleEndian, &o.LightInfo)
			if err != nil {
				return optByte, fmt.Errorf("error reading light info: %v", err)
			}
		} else if ef, ok := e.(*Effect); ok {
			err := binary.Read(r, binary.LittleEndian, &ef.LightInfo)
			if err != nil {
				return optByte, fmt.Errorf("error reading light info: %v", err)
			}
		} else if de, ok := e.(*DistanceEffect); ok {
			err := binary.Read(r, binary.LittleEndian, &de.LightInfo)
			if err != nil {
				return optByte, fmt.Errorf("error reading light info: %v", err)
			}
		} else {
			return optByte, fmt.Errorf("non-item/outfit/effect/distanceeffect lightcaster")
		}

	case 0x17: // Floor changing item.
		_, ok := e.(*Item)
		if !ok {
			return optByte, fmt.Errorf("non-item floor changer")
		}

	case 0x18: // Unknown information field.

	case 0x19: // Has offset.
		if i, ok := e.(*Item); ok {
			err := binary.Read(r, binary.LittleEndian, &i.OffsetInfo)
			if err != nil {
				return optByte, fmt.Errorf("error reading offset info: %v", err)
			}
		} else if o, ok := e.(*Outfit); ok {
			err := binary.Read(r, binary.LittleEndian, &o.OffsetInfo)
			if err != nil {
				return optByte, fmt.Errorf("error reading offset info: %v", err)
			}
		} else {
			return optByte, fmt.Errorf("non-item/outfit with offset (type %T)", e)
		}

	case 0x1A: // Player offset. Usually 8px
		i, ok := e.(*Item)
		if !ok {
			return optByte, fmt.Errorf("non-item with player offset")
		}
		err := binary.Read(r, binary.LittleEndian, &i.PlayerOffset)
		if err != nil {
			return optByte, fmt.Errorf("error reading player offset info: %v", err)
		}

	case 0x1B: // Draw with height offset for all parts of the sprite (usually a 2x2 block).
		i, ok := e.(*Item)
		if !ok {
			return optByte, fmt.Errorf("non-item with large height offset")
		}
		i.LargeOffset = true

	case 0x1C: // Animate while idling.
		if i, ok := e.(*Item); ok {
			i.IdleAnim = true
		} else if o, ok := e.(*Outfit); ok {
			o.IdleAnim = true
		} else {
			return optByte, fmt.Errorf("non-item/outfit idler (%T)", e)
		}

	case 0x1D: // Map color.
		i, ok := e.(*Item)
		if !ok {
			return optByte, fmt.Errorf("non-item with map color")
		}
		err := binary.Read(r, binary.LittleEndian, &i.MapColor)
		if err != nil {
			return optByte, fmt.Errorf("error reading map color info: %v", err)
		}

	case 0x1E: // Line spot.
		i, ok := e.(*Item)
		if !ok {
			return optByte, fmt.Errorf("non-item line spot")
		}
		var tmp uint8
		if err := binary.Read(r, binary.LittleEndian, &tmp); err != nil {
			return optByte, fmt.Errorf("error reading line spot info: %v", err)
		}
		if tmp == 0x58 {
			i.Readable = true
		} else if tmp != 0x4c && tmp != 0x4d && tmp != 0x4e && tmp != 0x4f && tmp != 0x50 && tmp != 0x51 && tmp != 0x52 && tmp != 0x53 && tmp != 0x54 && tmp != 0x55 && tmp != 0x56 && tmp != 0x57 {
			// 0x4c: can be used to go up
			// 0x4d: can be used to go down
			// 0x4e: unknown
			// 0x4f: switch
			// 0x50: unknown
			// 0x51: unknown
			// 0x52: stairs up
			// 0x54: unknown
			// 0x55: unknown
			// 0x56: openable holes
			// 0x57: unknown

			return optByte, fmt.Errorf("unknown linespot type 0x%x %d", tmp, tmp)
		}
		if err := binary.Read(r, binary.LittleEndian, &tmp); err != nil {
			return optByte, fmt.Errorf("error reading line spot info: %v", err)
		}
		if tmp != 0x04 {
			return optByte, fmt.Errorf("unknown linespot value 0x%x (expected 0x04)", tmp)
		}

	case 0x1F: // Unknown information field.

	case 0x20: // Look through.
		i, ok := e.(*Item)
		if !ok {
			return optByte, fmt.Errorf("non-item that can be looked-through")
		}
		i.LookThrough = true

	case 0xFF:
		return 0xFF, &endOfOptBlock{}

	default:
		return optByte, fmt.Errorf("unknown opt byte 0x%x", optByte)
	}
	return optByte, nil
}

// loadGraphicsSpec reads sprite information from the passed reader and stores it into the passed entry.
func (d *Dataset) loadGraphicsSpec(r io.Reader, e DatasetEntry) error {
	gfx := e.GetGraphics()

	if err := binary.Read(r, binary.LittleEndian, &gfx.GraphicsDimensions); err != nil {
		return fmt.Errorf("error reading graphics dimensions: %v", err)
	}
	glog.V(5).Info("reading rendersize")
	if gfx.GraphicsDimensions.Width != 1 || gfx.GraphicsDimensions.Height != 1 {
		if err := binary.Read(r, binary.LittleEndian, gfx.RenderSize); err != nil {
			return fmt.Errorf("error reading render size: %v", err)
		}

	}

	if err := binary.Read(r, binary.LittleEndian, &gfx.GraphicsDetails); err != nil {
		return fmt.Errorf("error reading graphics details: %v", err)
	}

	spriteCount := uint(gfx.GraphicsDimensions.Width * gfx.GraphicsDimensions.Height)
	spriteCount *= uint(gfx.GraphicsDetails.BlendFrames)
	spriteCount *= uint(gfx.GraphicsDetails.XDiv * gfx.GraphicsDetails.YDiv * gfx.GraphicsDetails.ZDiv)
	spriteCount *= uint(gfx.GraphicsDetails.AnimCount)
	if spriteCount == 0 {
		return fmt.Errorf("entry with zero sprites")
	}
	glog.V(5).Infof("allocating %d sprites", spriteCount)
	gfx.Sprites = make([]uint16, spriteCount)
	glog.V(5).Infof("reading %d sprites", spriteCount)

	if err := binary.Read(r, binary.LittleEndian, gfx.Sprites); err != nil {
		return fmt.Errorf("error reading sprites: %v", err)
	}
	glog.V(5).Infof("sprites have been read")
	return nil
}
