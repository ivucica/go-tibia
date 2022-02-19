package dat

import (
	"encoding/binary"
	"fmt"
	"io"
	"math"
	"strings"

	"github.com/golang/glog"
)

// Enumeration of versions that are 'actively' supported.
//
// (Values may change between versions of the package and have no meaning.)
const (
	CLIENT_VERSION_UNKNOWN = iota
	CLIENT_VERSION_854
)

// Field names in AppearanceFlag proto message in 12.x.
//
// TODO: find out where the names come from. They may be coming from otc
// c3deadf916 const.h or such; if so, they should only stay as names for
// proto fields. It's perhaps possible to come up with better names than
// either OTS or OTC (e.g. 'clip' likely refers to overlaid ground
// borders, but why 'bank' instead of ground? what's 'forceuse' vs
// 'usable'?)
//
// TODO: generate from the proto instead.
var protoFieldNames12x = []string{
	"", // 0, invalid

	"bank",                  // 1
	"clip",                  // 2
	"bottom",                // 3
	"top",                   // 4
	"container",             // 5
	"cumulative",            // 6
	"usable",                // 7
	"forceuse",              // 8
	"multiuse",              // 9
	"write",                 // 10
	"write_once",            // 11
	"liquidpool",            // 12
	"unpass",                // 13
	"unmove",                // 14
	"unsight",               // 15
	"avoid",                 // 16
	"no_movement_animation", // 17
	"take",                  // 18
	"liquidcontainer",       // 19
	"hang",                  // 20
	"hook",                  // 21
	"rotate",                // 22
	"light",                 // 23
	"dont_hide",             // 24
	"translucent",           // 25
	"shift",                 // 26
	"height",                // 27
	"lying_object",          // 28
	"animate_always",        // 29
	"automap",               // 30
	"lenshelp",              // 31
	"fullbank",              // 32
	"ignore_look",           // 33
	"clothes",               // 34
	"default_action",        // 35
	"market",                // 36
	"wrap",                  // 37
	"unwrap",                // 38
	"topeffect",             // 39
	"npcsaledata",           // 40
	"changedtoexpire",       // 41
	"corpse",                // 42
	"player_corpse",         // 43
	"cyclopediaitem",        // 44
}

type OptByte780 byte

// Note: proto field number mappings are temporary, and generally just 'i+1'.
// They need to be closely examined before they can be used and guaranteed
// a semantic match.
const (
	OptByte780Ground                = OptByte780(0x00) // 0, Proto name: bank
	OptByte780OnTop                 = OptByte780(0x01) // 1, Proto name: clip
	OptByte780WalkthroughItem       = OptByte780(0x02) // 2, Proto name: bottom
	OptByte780HigherWalkthroughItem = OptByte780(0x03) // 3, Proto name: top
	OptByte780Container             = OptByte780(0x04) // 4, Proto name: container
	OptByte780Stackable             = OptByte780(0x05) // 5, Proto name: cumulative
	OptByte780AlwaysUsed            = OptByte780(0x06) // 6, Proto name: usable
	OptByte780Usable                = OptByte780(0x07) // 7, Proto name: forceuse
	OptByte780Rune                  = OptByte780(0x08) // 8, Proto name: multiuse
	OptByte780RW                    = OptByte780(0x09) // 9, Proto name: write
	OptByte780RO                    = OptByte780(0x0A) // 10, Proto name: write_once
	OptByte780FluidContainer        = OptByte780(0x0B) // 11, Proto name: liquidpool
	OptByte780Splash                = OptByte780(0x0C) // 12, Proto name: unpass
	OptByte780BlockingPlayer        = OptByte780(0x0D) // 13, Proto name: unmove
	OptByte780ImmobileItem          = OptByte780(0x0E) // 14, Proto name: unsight
	OptByte780BlockingMissiles      = OptByte780(0x0F) // 15, Proto name: avoid
	OptByte780BlockingMonsters      = OptByte780(0x10) // 16, Proto name: no_movement_animation
	OptByte780Equipable             = OptByte780(0x11) // 16, Proto name: take
	OptByte780Hangable              = OptByte780(0x12) // 17, Proto name: liquidcontainer
	OptByte780HorizontalItem        = OptByte780(0x13) // 18, Proto name: hang
	OptByte780VerticalItem          = OptByte780(0x14) // 19, Proto name: hook
	OptByte780RotatableItem         = OptByte780(0x15) // 20, Proto name: rotate
	OptByte780Lightcaster           = OptByte780(0x16) // 21, Proto name: light
	OptByte780FloorChangingItem     = OptByte780(0x17) // 22, Proto name: dont_hide
	OptByte780Unknown0x18           = OptByte780(0x18) // 23, Proto name: translucent
	OptByte780HasOffset             = OptByte780(0x19) // 24, Proto name: shift
	OptByte780PlayerOffset          = OptByte780(0x1A) // 25, Proto name: height
	OptByte780HeightOffsetAllParts  = OptByte780(0x1B) // 26, Proto name: lying_object
	OptByte780IdleAnim              = OptByte780(0x1C) // 27, Proto name: animate_always
	OptByte780MapColor              = OptByte780(0x1D) // 28, Proto name: automap
	OptByte780LineSpot              = OptByte780(0x1E) // 29, Proto name: lenshelp
	OptByte780Unknown0x1F           = OptByte780(0x1F) // 30, Proto name: fullbank
	OptByte780LookThrough           = OptByte780(0x20) // 31, Proto name: ignore_look
	OptByte780Max                   = iota
)

// Note: proto field number mappings are temporary, and generally just 'i+1'.
// They need to be closely examined before they can be used and guaranteed
// a semantic match.
//
// TODO: add validFor validity bitmask, specifying if it's permitted for item,
// outfit, effect or distance effect, so validation is simplified.
var optByte780Names = []struct {
	constSuff, ot string
	b             byte
	protoFieldID  int
}{
	{"Ground", "ground tile", 0x00 /*0*/, 1},
	{"OnTop", "on-top items", 0x01 /*1*/, 2},
	{"WalkthroughItem", "walk-through items (e.g. doors)", 0x02 /*2*/, 3},
	{"HigherWalkthroughItem", "higher walk-through items (e.g. arches)", 0x03 /*3*/, 4},
	{"Container", "container item", 0x04 /*4*/, 5},
	{"Stackable", "stackable", 0x05 /*5*/, 6},
	{"AlwaysUsed", "always used (e.g. ladders)", 0x06 /*6*/, 7},
	{"Usable", "usable", 0x07 /*7*/, 8},
	{"Rune", "rune", 0x08 /*8*/, 9},
	{"RW", "R/W item", 0x09 /*9*/, 10},
	{"RO", "RO item", 0x0A /*10*/, 11},
	{"FluidContainer", "fluid container", 0x0B /*11*/, 12},
	{"Splash", "splash", 0x0C /*12*/, 13},
	{"BlockingPlayer", "blocking player", 0x0D /*13*/, 14},
	{"ImmobileItem", "immobile item", 0x0E /*14*/, 15},
	{"BlockingMissiles", "blocking missiles", 0x0F /*15*/, 16},
	{"BlockingMonsters", "blocking monsters", 0x10 /*16*/, 17},
	{"Equipable", "equipable", 0x11 /*17*/, 18},
	{"Hangable", "hangable", 0x12 /*18*/, 19},
	{"HorizontalItem", "horizontal item", 0x13 /*19*/, 20},
	{"VerticalItem", "vertical item", 0x14 /*20*/, 21},
	{"RotatableItem", "rotatable item", 0x15 /*21*/, 22},
	{"Lightcaster", "lightcaster", 0x16 /*22*/, 23},
	{"FloorChangingItem", "floor changing item", 0x17 /*23*/, 24},
	{"Unknown0x18", "unknown information field 0x18", 0x18 /*24*/, 25},
	{"HasOffset", "has offset", 0x19 /*25*/, 26},
	{"PlayerOffset", "player offset (usually 8px)", 0x1A /*26*/, 27},
	{"HeightOffsetAllParts", "draw with height offset for all parts of the sprite (usually a 2x2 block)", 0x1B /*27*/, 28},
	{"IdleAnim", "animate while idling", 0x1C /*28*/, 29},
	{"MapColor", "map color", 0x1D /*29*/, 30},
	{"LineSpot", "line spot", 0x1E /*30*/, 31},
	{"Unknown0x1F", "unknown information field 0x1f", 0x1F /*31*/, 32},
	{"LookThrough", "look through", 0x20 /*32*/, 33},
}

// OTStyleDescription returns a human readable description consisting of hex
// and dec representation of a byte, and a sentence-formatted short description
// of the byte.
func (b OptByte780) OTStyleDescription() string {
	if b >= OptByte780Max {
		return ""
	}
	ob := optByte780Names[b]
	d := ob.ot
	return fmt.Sprintf("0x%02x, %d: %s.", ob.b, ob.b, strings.ToTitle(d[:1])+d[1:])
}

// ConstName returns the name used for the constant in the source code.
func (b OptByte780) ConstName() string {
	if b >= OptByte780Max {
		return ""
	}
	return "OptByte780" + optByte780Names[b].constSuff
}

// ProtoFieldID returns 12.x's equivalent proto field number, or 0 if not
// mappable.
//
// Note: proto field number mappings are temporary, and generally just 'i+1'.
// They need to be closely examined before they can be used and guaranteed
// a semantic match.
func (b OptByte780) ProtoFieldID() int {
	if b >= OptByte780Max {
		return 0
	}
	return optByte780Names[b].protoFieldID
}

// ProtoFieldID returns 12.x's equivalent proto field name, or empty if not
// mappable.
//
// Note: proto field number mappings are temporary, and generally just 'i+1'.
// They need to be closely examined before they can be used and guaranteed
// a semantic match.
func (b OptByte780) ProtoFieldName() string {
	if b >= OptByte780Max {
		return ""
	}
	if optByte780Names[b].protoFieldID < 0 || optByte780Names[b].protoFieldID >= len(protoFieldNames12x) {
		return ""
	}
	return protoFieldNames12x[optByte780Names[b].protoFieldID]
}

// String returns a human-readable string detailing the information known about
// this byte: its hex and dec numbers, a short description, in-code enum const
// name, equivalent proto field id (if any) and the name (if any)
func (b OptByte780) String() string {
	if b.ProtoFieldID() > 0 && b.ProtoFieldID() < len(protoFieldNames12x) {
		return fmt.Sprintf("<0x%02x :: %s :: %s :: proto: %s = %d>", int(b), b.ConstName(), b.OTStyleDescription(), b.ProtoFieldName(), b.ProtoFieldID())
	}
	return fmt.Sprintf("<0x%02x :: %s :: %s>", int(b), b.ConstName(), b.OTStyleDescription())
}

// Dataset represents a set of items, outfits, effects and distance
// effects read from a Tibia dataset file ('things' or 'dataset entries').
type Dataset struct {
	Header

	items           []Item
	outfits         []Outfit
	effects         []Effect
	distanceEffects []DistanceEffect
}

type Header struct {
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
	MapColor     DatasetColor
	MapColorOK   bool
	LookThrough  bool

	// Raw bytes read from 7.80+ datafile.
	OptBytes780 []OptByte780
}

// Outfit represents a dataset entry describing a possible appearance for an in-game character, whether NPC or player.
type Outfit struct {
	DatasetEntry
	Graphics

	Id int
	OffsetInfo
	IdleAnim bool
	LightInfo

	// Raw bytes read from 7.80+ datafile.
	OptBytes780 []OptByte780
}

// Effect represents a temporarily-appearing in-game effect, such as a poof of smoke, then disappears.
type Effect struct {
	DatasetEntry
	Graphics

	Id int
	LightInfo

	// Raw bytes read from 7.80+ datafile.
	OptBytes780 []OptByte780
}

// DistanceEffect represents an in-game effect which moves from one tile to another over a period of time, then diappears.
type DistanceEffect struct {
	DatasetEntry
	Graphics

	Id int
	LightInfo

	// Raw bytes read from 7.80+ datafile.
	OptBytes780 []OptByte780
}

// DatasetColor fulfills the color.Color interface on top of the stored uint16 value.
//
// Usable for light color and map color.
type DatasetColor uint16

func (col DatasetColor) RGBA() (r, g, b, a uint32) {
	r = uint32(float64(col/36.) / 6. * math.MaxUint16)
	g = uint32(float64(uint32(float64(col)/6)%6) / 5. * math.MaxUint16)
	b = uint32(float64(col%6) / 5. * math.MaxUint16) // color.Color.RGBA() is defined to return in range [0, 0xFFFF]
	a = math.MaxUint16

	return

	// Base 6 per component. Values are 00, 33, 66, 99, cc, ff.
	// (We run out of colors by 0xD8.)
	//
	// color.Color.RGBA() is defined to return in range [0, 0xFFFF]
	lut := []uint32{
		0x00,
		0x33 << 8,
		0x66 << 8,
		0x99 << 8,
		0xcc << 8,
		0xff << 8,
	}

	origCol := col
	// Values 0 through 5 (base 6).
	b6 := col % 6
	col /= 6
	g6 := col % 6
	col /= 6
	r6 := col % 6

	r = lut[r6]
	g = lut[g6]
	b = lut[b6]
	a = math.MaxUint16

	glog.Infof("datasetcolor[%d / %02x] = %02x%02x%02x (%d %d %d)", origCol, origCol, r, g, b, r6, g6, b6)

	return
}

// LightInfo represents a recurring structure in the binary format of the data file concerning the strength and color of the light emitted client-side.
type LightInfo struct {
	Strength uint16
	Color    DatasetColor
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
func (i *Item) GetGraphics() *Graphics {
	return &i.Graphics
}

// String returns a string representation for an outfit.
func (o Outfit) String() string {
	return fmt.Sprintf("outfit %d", o.Id)
}

// GetGraphics returns sprites associated with this outfit.
func (o *Outfit) GetGraphics() *Graphics {
	return &o.Graphics
}

// String returns a string representation for an effect.
func (e Effect) String() string {
	return fmt.Sprintf("effect %d", e.Id)
}

// GetGraphics returns sprites associated with this effect.
func (e *Effect) GetGraphics() *Graphics {
	return &e.Graphics
}

// String returns a string representation for a distance effect.
func (d DistanceEffect) String() string {
	return fmt.Sprintf("distance effect %d", d.Id)
}

// GetGraphics returns sprites associated with this distance effect.
func (d *DistanceEffect) GetGraphics() *Graphics {
	return &d.Graphics
}

// NewDataset reads the dataset file from the passed io.Reader and returns the Dataset object.
func NewDataset(r io.Reader) (*Dataset, error) {
	glog.V(3).Info("starting to read dataset")
	h := Header{}
	if err := binary.Read(r, binary.LittleEndian, &h); err != nil {
		return nil, fmt.Errorf("error reading dataset header: %v", err)
	}

	glog.V(3).Infof("creating dataset")
	dataset := Dataset{
		Header:          h,
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

func (d *Dataset) Item(clientID uint16) *Item {
	if clientID < 100 {
		return nil
	}
	if int(clientID) > len(d.items)+100 {
		return nil
	}

	return &d.items[clientID-100]
}

func (d *Dataset) Outfit(clientID uint16) *Outfit {
	if d == nil {
		glog.Warningf("attempting to access outfit %d from a nil dataset", clientID)
		return nil
	}
	if clientID == 0 {
		glog.Warningf("attempting to access outfit 0 from dataset; clientIDs start with 1")
		return nil
	}
	if int(clientID) >= len(d.outfits)+1 {
		glog.Warningf("attempted to access outfit %d which doesn't exist in the dat", clientID)
		return nil
	}
	return &d.outfits[clientID-1]
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
		glog.V(4).Infof("gfx: %+v", e.GetGraphics())
		glog.V(4).Infoln("next item")
	}
	glog.V(3).Infof("done with 780 dataset")
	return nil
}

// ClientVersion returns which version of the game this data file comes from.
//
// Currently supported is only 8.54.
func (d Dataset) ClientVersion() int {
	if d.Header.Signature == 0x4b28b89e || d.Header.Signature == 0x4b1e2caa {
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
func (d *Dataset) load780OptByte(r io.Reader, entry DatasetEntry) (uint8, error) {
	var optByte uint8
	err := binary.Read(r, binary.LittleEndian, &optByte)
	if err != nil {
		return 0, fmt.Errorf("error reading the opt byte: %v", err)
	}

	var (
		i *Item
		o *Outfit
		e *Effect
		s *DistanceEffect // "shot"
	)

	switch v := entry.(type) {
	case *Item:
		i = v
		if optByte != 0xFF {
			i.OptBytes780 = append(i.OptBytes780, OptByte780(optByte))
		}
	case *Outfit:
		o = v
		if optByte != 0xFF {
			o.OptBytes780 = append(o.OptBytes780, OptByte780(optByte))
		}
	case *Effect:
		e = v
		if optByte != 0xFF {
			e.OptBytes780 = append(e.OptBytes780, OptByte780(optByte))
		}
	case *DistanceEffect:
		s = v
		if optByte != 0xFF {
			s.OptBytes780 = append(s.OptBytes780, OptByte780(optByte))
		}
	default:
		return 0, fmt.Errorf("unknown type of dataset entry (want item, outfit, effect or distanceeffect")
	}

	switch OptByte780(optByte) {
	case OptByte780Ground: // 0x00: Ground tile.
		if i == nil {
			return optByte, fmt.Errorf("non-item ground tile")
		}
		err := binary.Read(r, binary.LittleEndian, &i.GroundSpeed)
		if err != nil {
			return optByte, fmt.Errorf("error reading the ground tile speed: %v", err)
		}

	case OptByte780OnTop: // 0x01, 1: On-top items.
		if i == nil {
			return optByte, fmt.Errorf("non-item on-top entry")
		}
		i.SortOrder = 1

	case OptByte780WalkthroughItem: // 0x02, 2: Walk-through items (e.g. doors).
		if i == nil {
			return optByte, fmt.Errorf("non-item walk-through-1 entry")
		}
		i.SortOrder = 2

	case OptByte780HigherWalkthroughItem: // 0x03, 3: Higher walk-through items (e.g. arches).
		if i == nil {
			return optByte, fmt.Errorf("non-item walk-through-2 entry")
		}
		i.SortOrder = 3

	case OptByte780Container: // 0x04, 4: Container item.
		if i == nil {
			return optByte, fmt.Errorf("non-item container entry")
		}
		i.Container = true

	case OptByte780Stackable: // 0x05, 5: Stackable item.
		if i == nil {
			return optByte, fmt.Errorf("non-item stackable entry")
		}
		i.Stackable = true

	case OptByte780AlwaysUsed: // 0x06, 6: Always used. (e.g. ladders)
		if i == nil {
			return optByte, fmt.Errorf("non-item always-used entry")
		}
		i.AlwaysUsed = true

	case OptByte780Usable: // 0x07, 7: Usable.
		if i == nil {
			return optByte, fmt.Errorf("non-item usable entry")
		}
		i.Usable = true

	case OptByte780Rune: // 0x08, 8: Rune.
		if i == nil {
			return optByte, fmt.Errorf("non-item usable entry")
		}
		i.Rune = true

	case OptByte780RW: // 0x09, 9: R/W item.
		if i == nil {
			return optByte, fmt.Errorf("non-item RW item entry")
		}
		i.Readable = true
		i.Writable = true
		err := binary.Read(r, binary.LittleEndian, &i.MaxRWLen)
		if err != nil {
			return optByte, fmt.Errorf("error reading RW item info: %v", err)
		}

	case OptByte780RO: // 0x0A, 10: RO item.
		if i == nil {
			return optByte, fmt.Errorf("non-item RO item entry")
		}
		i.Readable = true
		err := binary.Read(r, binary.LittleEndian, &i.MaxRWLen)
		if err != nil {
			return optByte, fmt.Errorf("error reading RO item info: %v", err)
		}

	case OptByte780FluidContainer: // 0x0B, 11: Fluid container.
		if i == nil {
			return optByte, fmt.Errorf("non-item fluid container")
		}
		i.FluidContainer = true

	case OptByte780Splash: // 0x0C, 12: Splash.
		if i == nil {
			return optByte, fmt.Errorf("non-item splash")
		}
		i.Splash = true

	case OptByte780BlockingPlayer: // 0x0D, 13: Blocking player.
		if i == nil {
			return optByte, fmt.Errorf("non-item blocker")
		}
		i.BlockingPlayer = true

	case OptByte780ImmobileItem: // 0x0E, 14: Immobile item.
		if i == nil {
			return optByte, fmt.Errorf("non-item immobile")
		}
		i.Immobile = true

	case OptByte780BlockingMissiles: // 0x0F, 15: Blocking missiles.
		if i == nil {
			return optByte, fmt.Errorf("non-item blocker for missiles")
		}
		i.BlockingMissiles = true

	case OptByte780BlockingMonsters: // 0x10, 16: Blocking monsters.
		if i == nil {
			return optByte, fmt.Errorf("non-item blocker for monsters")
		}
		i.BlockingMonsters = true

	case OptByte780Equipable: // 0x11, 17: Equipable.
		if i == nil {
			return optByte, fmt.Errorf("non-item equipable")
		}
		i.Equipable = true

	case OptByte780Hangable: // 0x12, 18: Hangable.
		if i == nil {
			return optByte, fmt.Errorf("non-item equipable")
		}
		i.Hangable = true

	case OptByte780HorizontalItem: // 0x13, 19: Horizontal item.
		if i == nil {
			return optByte, fmt.Errorf("non-item horizontal")
		}
		i.HorizontalItem = true

	case OptByte780VerticalItem: // 0x14, 20: Vertical item.
		if i == nil {
			return optByte, fmt.Errorf("non-item vertical")
		}
		i.VerticalItem = true

	case OptByte780RotatableItem: // 0x15, 21: Rotatable item.
		if i == nil {
			return optByte, fmt.Errorf("non-item rotatable")
		}
		i.RotatableItem = true

	case OptByte780Lightcaster: // 0x16, 22: Lightcaster.
		var err error
		switch {
		case i != nil:
			err = binary.Read(r, binary.LittleEndian, &i.LightInfo)
		case o != nil:
			err = binary.Read(r, binary.LittleEndian, &o.LightInfo)
		case e != nil:
			err = binary.Read(r, binary.LittleEndian, &e.LightInfo)
		case s != nil:
			err = binary.Read(r, binary.LittleEndian, &s.LightInfo)
		default:
			return optByte, fmt.Errorf("non-item/outfit/effect/distanceeffect lightcaster")
		}
		if err != nil {
			return optByte, fmt.Errorf("error reading light info: %v", err)
		}

	case OptByte780FloorChangingItem: // 0x17, 23: Floor changing item.
		if i == nil {
			return optByte, fmt.Errorf("non-item floor changer")
		}

	case OptByte780Unknown0x18: // 0x18, 24: Unknown information field.

	case OptByte780HasOffset: // 0x19, 25: Has offset.
		var err error
		switch {
		case i != nil:
			err = binary.Read(r, binary.LittleEndian, &i.OffsetInfo)
		case o != nil:
			err = binary.Read(r, binary.LittleEndian, &o.OffsetInfo)
		default:
			return optByte, fmt.Errorf("non-item/outfit with offset (type %T)", e)
		}
		if err != nil {
			return optByte, fmt.Errorf("error reading offset info: %v", err)
		}

	case OptByte780PlayerOffset: // 0x1A, 26: Player offset. Usually 8px
		if i == nil {
			return optByte, fmt.Errorf("non-item with player offset")
		}
		err := binary.Read(r, binary.LittleEndian, &i.PlayerOffset)
		if err != nil {
			return optByte, fmt.Errorf("error reading player offset info: %v", err)
		}

	case OptByte780HeightOffsetAllParts: // 0x1B, 27: Draw with height offset for all parts of the sprite (usually a 2x2 block).
		if i == nil {
			return optByte, fmt.Errorf("non-item with large height offset")
		}
		i.LargeOffset = true

	case OptByte780IdleAnim: // 0x1C, 28: Animate while idling.
		switch {
		case i != nil:
			i.IdleAnim = true
		case o != nil:
			o.IdleAnim = true
		default:
			return optByte, fmt.Errorf("non-item/outfit idler (%T)", e)
		}

	case OptByte780MapColor: // 0x1D, 29: Map color.
		if i == nil {
			return optByte, fmt.Errorf("non-item with map color")
		}
		err := binary.Read(r, binary.LittleEndian, &i.MapColor)
		if err != nil {
			return optByte, fmt.Errorf("error reading map color info: %v", err)
		}
		i.MapColorOK = true

	case OptByte780LineSpot: // 0x1E, 30: Line spot.
		if i == nil {
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

	case OptByte780Unknown0x1F: // 0x1F, 31: Unknown information field.

	case OptByte780LookThrough: // 0x20, 32: Look through.
		if i == nil {
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
	gfx.RenderSize = 32
	if gfx.GraphicsDimensions.Width != 1 || gfx.GraphicsDimensions.Height != 1 {
		glog.V(5).Info("reading rendersize")
		if err := binary.Read(r, binary.LittleEndian, &gfx.RenderSize); err != nil {
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
