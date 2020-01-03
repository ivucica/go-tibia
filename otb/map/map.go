// Package otbm implements an OTBM map file format reader and a gameworld map data source.
package otbm

import (
	"badc0de.net/pkg/go-tibia/gameworld"
	tnet "badc0de.net/pkg/go-tibia/net"
	"badc0de.net/pkg/go-tibia/otb"
	"github.com/golang/glog"

	"badc0de.net/pkg/go-tibia/otb/items"
	"badc0de.net/pkg/go-tibia/things"

	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"strings"
)

type pos uint64

func (l pos) X() uint16 {
	return uint16(l & 0xFFFF)
}

func (l pos) Y() uint16 {
	return uint16((l >> 16) & 0xFFFF)
}

func (l pos) Floor() uint8 {
	return uint8((l >> 32) & 0xFF)
}

func (l pos) String() string {
	return fmt.Sprintf("(%d,%d,%d)", l.X(), l.Y(), l.Floor())
}

func posFromCoord(x, y uint16, floor uint8) pos {
	return pos((uint64(floor) << 32) | (uint64(y) << 16) | uint64(x))
}

type Map struct {
	otb.OTB
	gameworld.MapDataSource
	tiles     map[pos]*mapTile
	creatures map[gameworld.CreatureID]gameworld.Creature
	things    *things.Things

	defaultPlayerSpawnPoint pos // temporary variable; ideally this is specified by having player's town ID in config
}

func (m *Map) Private_And_Temp__DefaultPlayerSpawnPoint(c gameworld.CreatureID) tnet.Position {
	pos := m.defaultPlayerSpawnPoint
	return tnet.Position{
		X:     pos.X(),
		Y:     pos.Y(),
		Floor: pos.Floor(),
	}
}

type mapTile struct {
	gameworld.MapTile

	parent *Map

	ownPos pos

	ground    *mapItem
	layers    [][]*mapItem
	creatures []gameworld.Creature

	subscribers []gameworld.MapTileEventSubscriber
}

func (t *mapTile) String() string {
	return fmt.Sprintf("<tile at %s>", t.ownPos)
}

func (t *mapTile) GetItem(idx int) (gameworld.MapItem, error) {
	if t.ground != nil {
		if idx == 0 {
			return t.ground, nil
		} else {
			idx--
		}
	}
	for _, layer := range t.layers {
		if idx < len(layer) {
			return layer[idx], nil
		} else {
			idx -= len(layer)
		}
	}

	return nil, gameworld.ItemNotFound
}

func (t *mapTile) addItem(item *mapItem) error {
	// TODO notify of item updates (e.g. replacement)
	// for now, private method because it's used only during map load
	// maybe the public method will be a wrapper?

	m := t.parent

	if item.GetServerType() == 0 {
		glog.Warningf("   attempting to add item with server ID 0 to map; skipping")
		return nil
	}
	otbItem := m.things.Temp__GetItemFromOTB(item.GetServerType(), 0)
	if otbItem == nil {
		glog.Warningf("   OTB item %d cannot be found in the OTB items file.", item.GetServerType())
		return fmt.Errorf("otbm item %d not found in otb items", item.GetServerType())
	}
	if otbItem.Group == itemsotb.ITEM_GROUP_GROUND {
		if t.ground != nil {
			// maybe tell t.ground it is being replaced?
			// definitely notification will be different
			t.ground = item
		} else {
			t.ground = item
		}
		return nil
	}
	if len(t.layers) < 4 {
		// obviously an uninitialized tile.
		// TODO move to a 'makeTile' function on the map.
		t.layers = make([][]*mapItem, 4) // 0, 1, 2, 3
	}

	var ord uint8 // 0 by default
	if ordI, ok := otbItem.Attributes[itemsotb.ITEM_ATTR_TOPORDER]; ok {
		// 1: borders
		// 2: ladders, signs, splashes
		// 3: doors etc
		// beyond that, nonitems such as creatures which we don't store in layers

		ord = ordI.(uint8)
	}

	t.layers[ord] = append(t.layers[ord], item)

	return nil
}

func (t *mapTile) AddCreature(c gameworld.Creature) error {
	t.creatures = append(t.creatures, c)
	return nil
}

func (t *mapTile) RemoveCreature(cr gameworld.Creature) error {
	// not t.creatures - 1, in case the creature is not in fact stored on the tile.
	newCs := make([]gameworld.Creature, 0, len(t.creatures))
	seen := false
	for _, c := range t.creatures {
		if c.GetID() == cr.GetID() {
			seen = true
			newCs = append(newCs, c)
		}
	}
	if !seen {
		glog.Warningf("removing creature %d from tile %d %d %d where it's actually not present", cr.GetID(), cr.GetPos().X, cr.GetPos().Y, cr.GetPos().Floor)
	}
	return nil
}

func (t *mapTile) GetCreature(idx int) (gameworld.Creature, error) {
	if idx >= len(t.creatures) {
		return nil, gameworld.CreatureNotFound
	}
	return t.creatures[idx], nil
}

type mapTileArea struct {
	base pos
}

type mapItem struct {
	gameworld.MapItem

	parentTile  *mapTile
	parentItem  *mapItem
	ancestorMap *Map

	otbItemTypeID uint16

	count                int
	charges, runeCharges uint16
	actionID             uint16
	uniqueID             uint16
	depotID              uint16
	teleDest             pos
	text                 string
}

// GetServerType returns the server-side ID of the item.
func (i *mapItem) GetServerType() uint16 {
	return uint16(i.otbItemTypeID)
}

// GetCount returns the number of instances of this item (e.g. for coins).
//
// If item is unstackable, this will most likely be zero.
func (i *mapItem) GetCount() uint16 {
	return uint16(i.count)
}

func (i *mapItem) String() string {
	return fmt.Sprintf("<mapItem %d>", i.otbItemTypeID)
}

// Implementation detail: iota is not used primarily for easier referencing in
// case of an error.
type MapNodeType uint8

const (
	OTBM_ROOT        MapNodeType = 0x00
	OTBM_ROOTV1      MapNodeType = 0x01
	OTBM_MAP_DATA    MapNodeType = 0x02
	OTBM_ITEM_DEF    MapNodeType = 0x03
	OTBM_TILE_AREA   MapNodeType = 0x04
	OTBM_TILE        MapNodeType = 0x05
	OTBM_ITEM        MapNodeType = 0x06
	OTBM_TILE_SQUARE MapNodeType = 0x07
	OTBM_TILE_REF    MapNodeType = 0x08
	OTBM_SPAWNS      MapNodeType = 0x09
	OTBM_SPAWN_AREA  MapNodeType = 0x0A
	OTBM_MONSTER     MapNodeType = 0x0B
	OTBM_TOWNS       MapNodeType = 0x0C
	OTBM_TOWN        MapNodeType = 0x0D
	OTBM_HOUSETILE   MapNodeType = 0x0E
	OTBM_WAYPOINTS   MapNodeType = 0x0F
	OTBM_WAYPOINT    MapNodeType = 0x10
)

type ItemAttribute uint8

const (
	OTBM_ATTR_DESCRIPTION    ItemAttribute = 0x01
	OTBM_ATTR_EXT_FILE       ItemAttribute = 0x02
	OTBM_ATTR_TILE_FLAGS     ItemAttribute = 0x03
	OTBM_ATTR_ACTION_ID      ItemAttribute = 0x04
	OTBM_ATTR_UNIQUE_ID      ItemAttribute = 0x05
	OTBM_ATTR_TEXT           ItemAttribute = 0x06
	OTBM_ATTR_DESC           ItemAttribute = 0x07
	OTBM_ATTR_TELE_DEST      ItemAttribute = 0x08
	OTBM_ATTR_ITEM           ItemAttribute = 0x09
	OTBM_ATTR_DEPOT_ID       ItemAttribute = 0x0A
	OTBM_ATTR_EXT_SPAWN_FILE ItemAttribute = 0x0B
	OTBM_ATTR_RUNE_CHARGES   ItemAttribute = 0x0C
	OTBM_ATTR_EXT_HOUSE_FILE ItemAttribute = 0x0D
	OTBM_ATTR_HOUSEDOORID    ItemAttribute = 0x0E
	OTBM_ATTR_COUNT          ItemAttribute = 0x0F
	OTBM_ATTR_DURATION       ItemAttribute = 0x10
	OTBM_ATTR_DECAYING_STATE ItemAttribute = 0x11
	OTBM_ATTR_WRITTENDATE    ItemAttribute = 0x12
	OTBM_ATTR_WRITTENBY      ItemAttribute = 0x13
	OTBM_ATTR_SLEEPERGUID    ItemAttribute = 0x14
	OTBM_ATTR_SLEEPSTART     ItemAttribute = 0x15
	OTBM_ATTR_CHARGES        ItemAttribute = 0x16

	OTBM_ATTR_ATTRIBUTE_MAP ItemAttribute = 128
)

func (a ItemAttribute) String() string {
	switch a {
	case OTBM_ATTR_DESCRIPTION:
		return "description"
	case OTBM_ATTR_EXT_FILE:
		return "ext_file"
	case OTBM_ATTR_TILE_FLAGS:
		return "tile_flags"
	case OTBM_ATTR_ACTION_ID:
		return "action_id"
	case OTBM_ATTR_UNIQUE_ID:
		return "unique_id"
	case OTBM_ATTR_TEXT:
		return "text"
	case OTBM_ATTR_DESC:
		return "desc"
	case OTBM_ATTR_TELE_DEST:
		return "tele_dest"
	case OTBM_ATTR_ITEM:
		return "item"
	case OTBM_ATTR_DEPOT_ID:
		return "depot_id"
	case OTBM_ATTR_EXT_SPAWN_FILE:
		return "ext_spawn_file"
	case OTBM_ATTR_RUNE_CHARGES:
		return "rune_charges"
	case OTBM_ATTR_EXT_HOUSE_FILE:
		return "ext_house_file"
	case OTBM_ATTR_HOUSEDOORID:
		return "housedoorid"
	case OTBM_ATTR_COUNT:
		return "count"
	case OTBM_ATTR_DURATION:
		return "duration"
	case OTBM_ATTR_DECAYING_STATE:
		return "decaying_state"
	case OTBM_ATTR_WRITTENDATE:
		return "writtendate"
	case OTBM_ATTR_WRITTENBY:
		return "writtenby"
	case OTBM_ATTR_SLEEPERGUID:
		return "sleeperguid"
	case OTBM_ATTR_SLEEPSTART:
		return "sleepstart"
	case OTBM_ATTR_CHARGES:
		return "charges"

	case OTBM_ATTR_ATTRIBUTE_MAP:
		return "attribute_map"

	default:
		return fmt.Sprintf("unknown otbm attribute %02x", int(a))
	}
}

type rootHeader struct {
	Ver                          uint32
	Width, Height                uint16
	ItemsVerMajor, ItemsVerMinor uint32
}

// New reads an OTB file from a given reader.
func New(r io.ReadSeeker, t *things.Things) (*Map, error) {
	f, err := otb.NewOTB(r)
	if err != nil {
		return nil, fmt.Errorf("newotbm failed to use fileloader: %s", err)
	}

	otb := Map{
		OTB: *f,

		tiles:     map[pos]*mapTile{},
		creatures: map[gameworld.CreatureID]gameworld.Creature{},

		things: t,
	}

	root := otb.ChildNode(nil)
	if root == nil {
		return nil, fmt.Errorf("nil root node")
	}

	props := root.PropsBuffer()
	//var attr MapAttribute
	//if err := binary.Read(props, binary.LittleEndian, &attr); err != nil {
	//	return nil, fmt.Errorf("error reading otbm root node attr: %v", err)
	//}
	switch MapNodeType(root.NodeType()) {
	case OTBM_ROOT:
		var head rootHeader
		if err := binary.Read(props, binary.LittleEndian, &head); err != nil {
			return nil, fmt.Errorf("error reading otbm root node header attrs: %v", err)
		}

		glog.V(2).Infof("otbm header: %+v", head)
		// TODO: store version and ensure items.otb is applicable enough
	case OTBM_ROOTV1:
		return nil, fmt.Errorf("otbm with rootv1 header is not supported at this time")
	default:
		glog.Errorf("unknown root node 0x%02x", root.NodeType())
		return nil, fmt.Errorf("unknown root node 0x%02x", root.NodeType())
	}

	if otb.ChildNode(root) == nil {
		return nil, fmt.Errorf("no children in root node")
	}

	for node := otb.ChildNode(root); node != nil; node = node.NextNode() {
		if mapData, err := otb.readRootChildNode(node); err == nil {
			mapData = mapData // FIXME
		} else {
			return nil, fmt.Errorf("error reading root child node: %v", err)
		}
	}

	return &otb, nil
}

type MapData interface{}

// readRootChildNode reads a single "OTB node", as read from an OTB file.
func (m *Map) readRootChildNode(node *otb.OTBNode) (MapData, error) {

	switch MapNodeType(node.NodeType()) {
	case OTBM_MAP_DATA:
		return m.readMapDataNode(node)
	default:
		return nil, fmt.Errorf("readRootChildNode: unsupported node type 0x%02x", node.NodeType())
	}
}

func (m *Map) readMapDataNode(node *otb.OTBNode) (MapData, error) {
	// props := node.PropsBuffer() Likely nothing useful in PropsBuffer
	for node := m.ChildNode(node); node != nil; node = node.NextNode() {
		if mapData, err := m.readMapDataChildNode(node); err == nil {
			mapData = mapData
		} else {
			return nil, fmt.Errorf("error reading map data child node: %v", err)
		}
	}
	return nil, nil
}

func (m *Map) readMapDataChildNode(node *otb.OTBNode) (MapData, error) { // TODO: this won't return mapdata.
	switch MapNodeType(node.NodeType()) {
	case OTBM_ITEM_DEF:
		glog.V(2).Infof("item definition")
	case OTBM_TILE_AREA:
		return m.readTileAreaNode(node)
	case OTBM_TOWNS:
		return m.readTownsNode(node)
	case OTBM_WAYPOINTS:
		glog.V(2).Infof("waypoints")
	default:
		return nil, fmt.Errorf("readMapDataChildNode: unsupported node type 0x%02x", node.NodeType())
	}
	return nil, nil
}

func (m *Map) readTileAreaNode(node *otb.OTBNode) (MapData, error) { // TODO: this won't return mapdata.
	propBuf := node.PropsBuffer()
	type propType struct {
		X, Y  uint16
		Floor uint8
	}
	props := propType{}

	if err := binary.Read(propBuf, binary.LittleEndian, &props); err != nil {
		return nil, fmt.Errorf("error reading props of tile area node: %v", err)
	}

	area := mapTileArea{
		base: posFromCoord(props.X, props.Y, props.Floor),
	}

	glog.V(2).Infof("tile area at %d,%d,%d", props.X, props.Y, props.Floor)

	for node := m.ChildNode(node); node != nil; node = node.NextNode() {
		if mapData, err := m.readTileAreaChildNode(node, &area); err == nil {
			mapData = mapData
		} else {
			return nil, fmt.Errorf("error reading tile area child node: %v", err)
		}
	}
	return nil, nil
}

func (m *Map) readTileAreaChildNode(node *otb.OTBNode, area *mapTileArea) (MapData, error) { // TODO: this won't return mapdata.
	switch MapNodeType(node.NodeType()) {
	case OTBM_TILE:
		return m.readTileNode(node, area)
	case OTBM_HOUSETILE:
		glog.V(2).Infof(" house tile")
	default:
		return nil, fmt.Errorf("readTileAreaChildNode: unsupported node type 0x%02x", node.NodeType())
	}
	return nil, nil
}

func (m *Map) readTileNode(node *otb.OTBNode, area *mapTileArea) (MapData, error) { // TODO: this won't return mapdata.
	propBuf := node.PropsBuffer()
	type propType struct {
		X, Y uint8
	}
	props := propType{}

	if err := binary.Read(propBuf, binary.LittleEndian, &props); err != nil {
		return nil, fmt.Errorf("error reading props of tile node: %v", err)
	}

	p := posFromCoord(area.base.X()+uint16(props.X), area.base.Y()+uint16(props.Y), area.base.Floor())
	tile := mapTile{ownPos: p, parent: m}
	m.tiles[p] = &tile

	glog.V(2).Infof(" tile at %d,%d,%d (%d+%d,%d+%d,%d)", p.X(), p.Y(), p.Floor(), area.base.X(), props.X, area.base.Y(), props.Y, area.base.Floor())

	for attr, err := propBuf.ReadByte(); err == nil; attr, err = propBuf.ReadByte() {
		switch ItemAttribute(attr) {
		case OTBM_ATTR_TILE_FLAGS:
			var tileFlags uint32
			if err := binary.Read(propBuf, binary.LittleEndian, &tileFlags); err != nil {
				return nil, fmt.Errorf("readTileNode: error reading flags attr of tile: %v", err)
			}
			glog.V(2).Infof("  tileflags: %04x", tileFlags)
		case OTBM_ATTR_ITEM:
			item := &mapItem{
				ancestorMap: m,
				parentTile:  &tile,
				count:       1,
			}
			if err := binary.Read(propBuf, binary.LittleEndian, &item.otbItemTypeID); err != nil {
				return nil, fmt.Errorf("readTileNode: error reading item prop of tile: %v", err)
			}

			// if otbm version is MAP_OTBM_1 and item is stackable, or splash, or fluid container, read one more byte
			// TODO: check for otbm_1
			otbItem := m.things.Temp__GetItemFromOTB(item.GetServerType(), 0)
			if otbItem.Group == itemsotb.ITEM_GROUP_SPLASH || otbItem.Group == itemsotb.ITEM_GROUP_FLUID || otbItem.Flags&itemsotb.FLAG_STACKABLE != 0 {
				cntB, err := propBuf.ReadByte()
				if err != nil {
					return nil, fmt.Errorf("readTileNode: countable item error: %v", err)
				}
				glog.V(2).Infof("    -> count %d", cntB)
				item.count = int(cntB)
			}

			tile.addItem(item)
			glog.V(2).Infof("  tileitem: %04x", item.otbItemTypeID)
		default:
			return nil, fmt.Errorf("readTileNode: unsupported attr type 0x%02x", attr)
		}
	}

	for node := m.ChildNode(node); node != nil; node = node.NextNode() {
		if mapData, err := m.readTileChildNode(node, &tile); err == nil {
			mapData = mapData
		} else {
			return nil, fmt.Errorf("error reading tile child node: %v", err)
		}
	}
	return nil, nil
}

func (m *Map) readTileChildNode(node *otb.OTBNode, tile *mapTile) (MapData, error) { // TODO: this won't return mapdata.
	switch MapNodeType(node.NodeType()) {
	case OTBM_ITEM:
		return m.readItemNode(node, tile, nil, 2)
	default:
		return nil, fmt.Errorf("readTileChildNode: unsupported node type 0x%02x", node.NodeType())
	}
	return nil, nil
}

func (m *Map) readItemNode(node *otb.OTBNode, parentTile *mapTile, parentItem *mapItem, depth int) (MapData, error) { // TODO: this won't return mapdata.
	glog.V(2).Infof("%sitem", strings.Repeat(" ", depth))
	propBuf := node.PropsBuffer()

	item := &mapItem{
		ancestorMap: m,
		parentTile:  parentTile,
		parentItem:  parentItem,
	}

	if err := binary.Read(propBuf, binary.LittleEndian, &item.otbItemTypeID); err != nil {
		return nil, fmt.Errorf("error reading prop otbItemTypeID of item node: %v", err)
	}

	// TODO: read other props

	for node := m.ChildNode(node); node != nil; node = node.NextNode() {
		if mapData, err := m.readItemChildNode(node, parentTile, item, depth+1); err == nil {
			mapData = mapData
		} else {
			return nil, fmt.Errorf("error reading tile child node: %v", err)
		}
	}

	if parentItem != nil {
		// TODO: parentItem.AddChild(...)
	} else if parentTile != nil {
		//otbItem := m.things.Temp__GetItemFromOTB(item.GetServerType(), 0)
		//if otbItem.Group == itemsotb.ITEM_GROUP_GROUND {
		//}
		parentTile.addItem(item)
	}

	return nil, nil
}

func (m *Map) readItemChildNode(node *otb.OTBNode, parentTile *mapTile, parentItem *mapItem, depth int) (MapData, error) { // TODO: this won't return mapdata.
	switch MapNodeType(node.NodeType()) {
	case OTBM_ITEM:
		return m.readItemNode(node, parentTile, parentItem, depth)
	default:
		return nil, fmt.Errorf("readItemChildNode: unsupported node type 0x%02x", node.NodeType())
	}
	return nil, nil
}

func (m *Map) readTownsNode(node *otb.OTBNode) (MapData, error) { // TODO: this won't return mapdata.
	for node := m.ChildNode(node); node != nil; node = node.NextNode() {
		if mapData, err := m.readTownsChildNode(node); err == nil {
			mapData = mapData
		} else {
			return nil, fmt.Errorf("error reading towns child node: %v", err)
		}
	}
	return nil, nil
}

func (m *Map) readTownsChildNode(node *otb.OTBNode) (MapData, error) { // TODO: this won't return mapdata.
	switch MapNodeType(node.NodeType()) {
	case OTBM_TOWN:
		return m.readTownNode(node)
	default:
		return nil, fmt.Errorf("readTownNode: unsupported node type 0x%02x", node.NodeType())
	}
	return nil, nil
}

func (m *Map) readTownNode(node *otb.OTBNode) (MapData, error) { // TODO: this won't return mapdata.
	propBuf := node.PropsBuffer()
	type propType struct {
		id        uint32
		name      string
		templePos struct {
			TempleX, TempleY uint16
			TempleFloor      uint8
		}
	}
	props := propType{}

	if err := binary.Read(propBuf, binary.LittleEndian, &props.id); err != nil {
		return nil, fmt.Errorf("error reading prop id of town node: %v", err)
	}

	var nameSize uint16
	if err := binary.Read(propBuf, binary.LittleEndian, &nameSize); err != nil {
		return nil, fmt.Errorf("error reading prop name's size in town node: %v", err)
	}

	nameB := make([]byte, nameSize)
	n, err := propBuf.Read(nameB)
	if err != nil {
		return nil, fmt.Errorf("error reading prop name of town node: %v", err)
	}
	if n != int(nameSize) {
		return nil, fmt.Errorf("did not read entire name in town node: got %d, want %d", n, nameSize)
	}
	props.name = string(nameB) // assume utf8, I suppose

	if err := binary.Read(propBuf, binary.LittleEndian, &props.templePos); err != nil {
		return nil, fmt.Errorf("error reading prop templePos of town node: %v", err)
	}

	glog.V(2).Infof(" town %s (%d) with temple at %d,%d,%d", props.name, props.id, props.templePos.TempleX, props.templePos.TempleY, props.templePos.TempleFloor)

	if m.defaultPlayerSpawnPoint == 0 {
		m.defaultPlayerSpawnPoint = posFromCoord(props.templePos.TempleX, props.templePos.TempleY, props.templePos.TempleFloor)
		glog.V(2).Infof("  this town is now the default spawn point %v", m.defaultPlayerSpawnPoint)
	}

	// skipping child nodes

	return nil, nil
}

func (m *Map) AddCreature(c gameworld.Creature) error {
	glog.V(2).Infof("adding creature %d", c.GetID())
	m.creatures[c.GetID()] = c
	if t, err := m.GetMapTile(c.GetPos().X, c.GetPos().Y, c.GetPos().Floor); err != nil {
		return err
	} else {
		glog.V(2).Infof("adding creature to %d %d %d", c.GetPos().X, c.GetPos().Y, c.GetPos().Floor)

		// HACK: tile has no ground? add it.
		// REMOVE THIS once maps are correctly loaded.
		if i, err := t.GetItem(0); err != nil || i == nil {
			glog.V(2).Info("  but first adding some ground for the creature")
			item := &mapItem{
				ancestorMap:   m,
				parentTile:    t.(*mapTile),
				parentItem:    nil,
				otbItemTypeID: 100,
			}
			t.(*mapTile).ground = item
		}

		return t.AddCreature(c)
	}
}

func (m *Map) GetMapTile(x, y uint16, z uint8) (gameworld.MapTile, error) {
	pos := posFromCoord(x, y, z)
	if t, ok := m.tiles[pos]; ok { //tnet.Position{x, y, z}]; ok {
		return t, nil
	}
	//return nil, fmt.Errorf("tile not found")
	return &mapTile{parent: m}, nil
}

func (m *Map) GetCreatureByIDBytes(idBytes [4]byte) (gameworld.Creature, error) {
	buf := bytes.NewBuffer(idBytes[:])
	var id gameworld.CreatureID
	err := binary.Read(buf, binary.LittleEndian, &id)
	if err != nil {
		return nil, fmt.Errorf("could not decode creature ID from bytes: %v", err)
	}

	return m.GetCreatureByID(id)
}
func (m *Map) GetCreatureByID(id gameworld.CreatureID) (gameworld.Creature, error) {
	if creature, ok := m.creatures[id]; ok {
		return creature, nil
	}
	return nil, gameworld.CreatureNotFound
}

func (m *Map) RemoveCreatureByID(id gameworld.CreatureID) error {
	c, err := m.GetCreatureByID(id)
	if err != nil {
		if err == gameworld.CreatureNotFound {
			return nil
		}
	}

	delete(m.creatures, id)

	if t, err := m.GetMapTile(c.GetPos().X, c.GetPos().Y, c.GetPos().Floor); err != nil {
		return err
	} else {
		glog.V(2).Infof("deleting creature from %d %d %d", c.GetPos().X, c.GetPos().Y, c.GetPos().Floor)
		return t.RemoveCreature(c)
	}
}
