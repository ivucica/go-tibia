// Package otbm implements an OTBM map file format reader and a gameworld map data source.
package otbm

import (
	"badc0de.net/pkg/go-tibia/gameworld"
	"badc0de.net/pkg/go-tibia/otb"
	"github.com/golang/glog"

	"badc0de.net/pkg/go-tibia/otb/items"

	"encoding/binary"
	"fmt"
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

type mapTile struct {
	gameworld.MapTile

	parent *Map

	ownPos pos

	ground    mapItem
	layers    [][]*mapItem
	creatures []gameworld.Creature

	subscribers []gameworld.MapTileEventSubscriber
}

func (t *mapTile) String() string {
	return fmt.Sprintf("<tile at %s; %d creatures>", t.ownPos, len(t.creatures))
}

func (t *mapTile) GetItem(idx int) (gameworld.MapItem, error) {
	if t.ground.otbItemTypeID != 0 {
		if idx == 0 {
			return &t.ground, nil
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

func (t *mapTile) addItem(item mapItem) error {
	// TODO notify of item updates (e.g. replacement)
	// for now, private method because it's used only during map load
	// maybe the public method will be a wrapper?

	m := t.parent

	if item.GetServerType() == 0 {
		glog.Warningf("   attempting to add item with server ID 0 to map tile %s; skipping", t.String())
		return nil
	}
	otbItem := m.things.Temp__GetItemFromOTB(item.GetServerType(), 0)
	if otbItem == nil {
		glog.Warningf("   OTB item %d cannot be found in the OTB items file.", item.GetServerType())
		return fmt.Errorf("otbm item %d not found in otb items", item.GetServerType())
	}
	if otbItem.Group == itemsotb.ITEM_GROUP_GROUND {
		if t.ground.otbItemTypeID != 0 {
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

	t.layers[ord] = append(t.layers[ord], &item)

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
		} else {
			glog.Warningf("seeing creature %d at %v; looking for %d", c.GetID(), cr.GetPos(), cr.GetID())
			newCs = append(newCs, c)
		}
	}
	if !seen {
		glog.Warningf("removing creature %d from tile %d %d %d where it's actually not present", cr.GetID(), cr.GetPos().X, cr.GetPos().Y, cr.GetPos().Floor)
	}
	t.creatures = newCs
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
	name := "unnamed"
	clientID := uint16(0)
	if i.GetServerType() != 0 {
		otbItem := i.ancestorMap.things.Temp__GetItemFromOTB(i.GetServerType(), 0)
		if otbItem != nil {
			name = otbItem.Name()
			clientID = i.ancestorMap.things.Temp__GetClientIDForServerID(i.GetServerType(), 0)
		}
	}

	return fmt.Sprintf("<mapItem %d : %02x %s>", i.otbItemTypeID, clientID, name)
}

type rootHeader struct {
	Ver                          uint32
	Width, Height                uint16
	ItemsVerMajor, ItemsVerMinor uint32
}

// readRootChildNode reads a single "OTB node", as read from an OTB file.
func (m *Map) readRootChildNode(node *otb.OTBNode) error {

	switch MapNodeType(node.NodeType()) {
	case OTBM_MAP_DATA:
		return m.readMapDataNode(node)
	default:
		return fmt.Errorf("readRootChildNode: unsupported node type 0x%02x", node.NodeType())
	}
}

func (m *Map) readMapDataNode(node *otb.OTBNode) error {
	propBuf := node.PropsBuffer()

	readStr := func() (string, error) {
		var sz uint16
		if err := binary.Read(propBuf, binary.LittleEndian, &sz); err != nil {
			return "", err
		}
		buf := make([]byte, sz)
		n, err := propBuf.Read(buf)
		if err != nil {
			return "", err
		}
		if n != int(sz) {
			return "", fmt.Errorf("sz %d != n %d", sz, n)
		}
		return string(buf), nil
	}

	for attr, err := propBuf.ReadByte(); err == nil; attr, err = propBuf.ReadByte() {
		attr := ItemAttribute(attr)
		switch attr {
		case OTBM_ATTR_DESCRIPTION:
			s, err := readStr()
			if err != nil {
				return fmt.Errorf("bad attr description in node: %w", err)
			}
			m.desc = append(m.desc, s)
		case OTBM_ATTR_EXT_SPAWN_FILE:
			s, err := readStr()
			if err != nil {
				return fmt.Errorf("bad attr ext spawn file in node: %w", err)
			}
			m.extSpawnFile = s
		case OTBM_ATTR_EXT_HOUSE_FILE:
			s, err := readStr()
			if err != nil {
				return fmt.Errorf("bad attr ext house file in node: %w", err)
			}
			m.extHouseFile = s
		default:
			return fmt.Errorf("readMapData: unsupported attr type 0x%02x (%s)", attr, attr)
		}
	}

	glog.Infof("Reading map %s", m)
	for node := node.ChildNode(); node != nil; node = node.NextNode() {
		if err := m.readMapDataChildNode(node); err != nil {
			return fmt.Errorf("error reading map data child node: %w", err)
		}
	}
	return nil
}

func (m *Map) readMapDataChildNode(node *otb.OTBNode) error {
	switch MapNodeType(node.NodeType()) {
	case OTBM_ITEM_DEF:
		glog.V(2).Infof("item definition")
	case OTBM_TILE_AREA:
		return m.readTileAreaNode(node)
	case OTBM_TOWNS:
		return m.readTownsNode(node)
	case OTBM_WAYPOINTS:
		return m.readWaypointsNode(node)
	default:
		return fmt.Errorf("readMapDataChildNode: unsupported node type 0x%02x", node.NodeType())
	}
	return nil
}

func (m *Map) readTileAreaNode(node *otb.OTBNode) error {
	propBuf := node.PropsBuffer()
	type propType struct {
		X, Y  uint16
		Floor uint8
	}
	props := propType{}

	if err := binary.Read(propBuf, binary.LittleEndian, &props); err != nil {
		return fmt.Errorf("error reading props of tile area node: %v", err)
	}

	area := mapTileArea{
		base: posFromCoord(props.X, props.Y, props.Floor),
	}

	if glog.V(2) {
		glog.Infof("tile area at %d,%d,%d for %v", props.X, props.Y, props.Floor, m)
	}

	for node := node.ChildNode(); node != nil; node = node.NextNode() {
		if err := m.readTileAreaChildNode(node, &area); err != nil {
			return fmt.Errorf("error reading tile area child node: %v", err)
		}
	}
	return nil
}

func (m *Map) readTileAreaChildNode(node *otb.OTBNode, area *mapTileArea) error {
	switch nt := MapNodeType(node.NodeType()); nt {
	case OTBM_HOUSETILE:
		fallthrough
	case OTBM_TILE:
		return m.readTileOrHouseTileNode(node, area, nt)
	default:
		return fmt.Errorf("readTileAreaChildNode: unsupported node type 0x%02x", node.NodeType())
	}
	return nil
}

func (m *Map) readTileOrHouseTileNode(node *otb.OTBNode, area *mapTileArea, nt MapNodeType) error {
	propBuf := node.PropsBuffer()
	type propType struct {
		X, Y uint8
	}
	props := propType{}

	if err := binary.Read(propBuf, binary.LittleEndian, &props); err != nil {
		return fmt.Errorf("error reading props of tile node: %v", err)
	}

	p := posFromCoord(area.base.X()+uint16(props.X), area.base.Y()+uint16(props.Y), area.base.Floor())
	tile := mapTile{ownPos: p, parent: m}
	m.tiles[p] = &tile

	var v glog.Level
	v = 2

	if glog.V(v) {
		glog.Infof(" tile at %d,%d,%d (%d+%d,%d+%d,%d) of %v", p.X(), p.Y(), p.Floor(), area.base.X(), props.X, area.base.Y(), props.Y, area.base.Floor(), m)
	}
	defer func() {
		if glog.V(v) {
			glog.Infof(" end tile at %d,%d,%d (%d+%d,%d+%d,%d) of %v", p.X(), p.Y(), p.Floor(), area.base.X(), props.X, area.base.Y(), props.Y, area.base.Floor(), m)
		}
	}()

	if nt == OTBM_HOUSETILE {
		var houseID uint32
		if err := binary.Read(propBuf, binary.LittleEndian, &houseID); err != nil {
			return fmt.Errorf("readTileNode: error reading flags attr of tile: %v", err)
		}
		glog.V(v).Infof("  house ID: %04x", houseID)
	}

	for attr, err := propBuf.ReadByte(); err == nil; attr, err = propBuf.ReadByte() {
		attr := ItemAttribute(attr)
		switch attr {
		case OTBM_ATTR_TILE_FLAGS:
			var tileFlags uint32
			if err := binary.Read(propBuf, binary.LittleEndian, &tileFlags); err != nil {
				return fmt.Errorf("readTileNode: error reading flags attr of tile: %v", err)
			}
			glog.V(v).Infof("  tileflags: %04x", tileFlags)
		case OTBM_ATTR_ITEM:
			item := mapItem{
				ancestorMap: m,
				parentTile:  &tile,
				count:       1,
			}
			if err := binary.Read(propBuf, binary.LittleEndian, &item.otbItemTypeID); err != nil {
				return fmt.Errorf("readTileNode: error reading item prop of tile: %v", err)
			}
			if glog.V(v) {
				glog.Infof("  tileitem: %02d %04x", item.otbItemTypeID, item.otbItemTypeID)
			}

			// if otbm version is MAP_OTBM_1 and item is stackable, or splash, or fluid container, read one more byte
			// TODO: check for otbm_1
			otbItem := m.things.Temp__GetItemFromOTB(item.GetServerType(), 0)
			if otbItem == nil {
				glog.Errorf("could not get otb for item %d!", item.GetServerType())
				continue
			}
			// n.b. Item group is supposed to be ground here!
			if glog.V(v) {
				glog.Infof("  item group: %s", otbItem.Group)
			}
			if glog.V(v) {
				glog.Infof("  item flags: %s", otbItem.Flags)
			}
			if otbItem.Group == itemsotb.ITEM_GROUP_SPLASH || otbItem.Group == itemsotb.ITEM_GROUP_FLUID || otbItem.Flags&itemsotb.FLAG_STACKABLE != 0 {
				cntB, err := propBuf.ReadByte()
				if err != nil {
					return fmt.Errorf("readTileNode: countable item error: %v", err)
				}
				if glog.V(v) {
					glog.Infof("    -> count %d", cntB)
				}
				panic(cntB)
				item.count = int(cntB)
			}

			tile.addItem(item)
		default:
			return fmt.Errorf("readTileNode: unsupported attr type 0x%02x (%s)", attr, attr)
		}
	}

	for node := node.ChildNode(); node != nil; node = node.NextNode() {
		if err := m.readTileChildNode(node, &tile); err != nil {
			return fmt.Errorf("error reading tile child node: %v", err)
		}
	}
	return nil
}

func (m *Map) readTileChildNode(node *otb.OTBNode, tile *mapTile) error {
	switch MapNodeType(node.NodeType()) {
	case OTBM_ITEM:
		return m.readItemNode(node, tile, nil, 2)
	default:
		return fmt.Errorf("readTileChildNode: unsupported node type 0x%02x", node.NodeType())
	}
	return nil
}

func (m *Map) readItemNode(node *otb.OTBNode, parentTile *mapTile, parentItem *mapItem, depth int) error {
	var indent string
	if glog.V(2) {
		indent = strings.Repeat(" ", depth+1)
	}

	if glog.V(2) {
		glog.Infof("%sitem", indent)
	}
	propBuf := node.PropsBuffer()

	item := mapItem{
		ancestorMap: m,
		parentTile:  parentTile,
		parentItem:  parentItem,
	}

	if err := binary.Read(propBuf, binary.LittleEndian, &item.otbItemTypeID); err != nil {
		return fmt.Errorf("error reading prop otbItemTypeID of item node: %v", err)
	}

	v := glog.Level(2)
	glog.V(v).Infof("%sitem id: %d", indent, item.otbItemTypeID)
	if item.otbItemTypeID != 0 {
		otbItem := m.things.Temp__GetItemFromOTB(item.GetServerType(), 0)
		if otbItem != nil {
			if glog.V(v) {
				glog.Infof("%sitem name: %s", indent, otbItem.Name())
				glog.Infof("%sitem group: %s", indent, otbItem.Group)
				glog.Infof("%sitem flags: %s", indent, otbItem.Flags)
			}
			// if otbm version is MAP_OTBM_1 and item is stackable, or splash, or fluid container, read one more byte
			// TODO: check for otbm_1
			if otbItem.Group == itemsotb.ITEM_GROUP_SPLASH || otbItem.Group == itemsotb.ITEM_GROUP_FLUID || otbItem.Flags&itemsotb.FLAG_STACKABLE != 0 {
			}
		} else {
			if glog.V(v) {
				glog.Infof("%s[n.b. item nil in items.otb]", indent)
			}
		}
	} else {
		//glog.Errorf("%s[n.b. item id on map is 0]", indent)
		//v = 0
		return nil // just ignore the item for the time being, figure out what's up later...
	}

	for attr, err := propBuf.ReadByte(); err == nil; attr, err = propBuf.ReadByte() {
		attr := ItemAttribute(attr)
		switch attr {
		case OTBM_ATTR_COUNT:
			cntB, err := propBuf.ReadByte()
			if err != nil {
				return fmt.Errorf("readItemNode: countable item error: %v", err)
			}
			if glog.V(v) {
				glog.Infof("%sitem count: %d", indent, cntB)
			}
			item.count = int(cntB)
		case OTBM_ATTR_RUNE_CHARGES:
			if err := binary.Read(propBuf, binary.LittleEndian, &item.runeCharges); err != nil {
				return fmt.Errorf("readItemNode: rune item error: %v", err)
			}
			if glog.V(v) {
				glog.Infof("%sitem rune charges: %d", indent, item.runeCharges)
			}
		case OTBM_ATTR_CHARGES:
			if err := binary.Read(propBuf, binary.LittleEndian, &item.charges); err != nil {
				return fmt.Errorf("readItemNode: chargable item error: %v", err)
			}
			if glog.V(v) {
				glog.Infof("%sitem charges: %d", indent, item.charges)
			}
		case OTBM_ATTR_ACTION_ID:
			if err := binary.Read(propBuf, binary.LittleEndian, &item.actionID); err != nil {
				return fmt.Errorf("readItemNode: actionable item error: %v", err)
			}
			if glog.V(v) {
				glog.Infof("%sitem action ID: %d", indent, item.actionID)
			}
		case OTBM_ATTR_UNIQUE_ID:
			if err := binary.Read(propBuf, binary.LittleEndian, &item.uniqueID); err != nil {
				return fmt.Errorf("readItemNode: unique item error: %v", err)
			}
			if glog.V(v) {
				glog.Infof("%sitem unique ID: %d", indent, item.uniqueID)
			}
		case OTBM_ATTR_DEPOT_ID:
			if err := binary.Read(propBuf, binary.LittleEndian, &item.depotID); err != nil {
				return fmt.Errorf("readItemNode: depotid item error: %v", err)
			}
			if glog.V(v) {
				glog.Infof("%sitem depot ID: %d", indent, item.depotID)
			}
		case OTBM_ATTR_TELE_DEST:
			var teleDest struct {
				X, Y  uint16
				Floor uint8
			}
			if err := binary.Read(propBuf, binary.LittleEndian, &teleDest); err != nil {
				return fmt.Errorf("readItemNode: teledest item error: %v", err)
			}
			item.teleDest = posFromCoord(teleDest.X, teleDest.Y, teleDest.Floor)
			if glog.V(v) {
				glog.Infof("%sitem teledest: %s", indent, item.teleDest)
			}
		case OTBM_ATTR_TEXT:
			var textSize uint16
			if err := binary.Read(propBuf, binary.LittleEndian, &textSize); err != nil {
				return fmt.Errorf("readItemNode: texted item error: %v", err)
			}
			textB := make([]byte, textSize)
			n, err := propBuf.Read(textB)
			if err != nil {
				return fmt.Errorf("error reading prop text of item node: %v", err)
			}
			if n != int(textSize) {
				return fmt.Errorf("did not read entire text in item node: got %d, want %d", n, textSize)
			}
			item.text = string(textB) // assume utf8, I suppose

			if glog.V(v) {
				glog.Infof("%sitem text[%d]: %s", indent, textSize, item.text)
			}
		case OTBM_ATTR_HOUSEDOORID:
			var houseDoorID [1]byte

			n, err := propBuf.Read(houseDoorID[:])
			if err != nil {
				return fmt.Errorf("failed to read house door id: %v", err)
			}
			if n != 1 {
				return fmt.Errorf("failed to read house door id, got only %d bytes", n)
			}

			if glog.V(v) {
				glog.Infof("%shouse door id: %d", indent, houseDoorID[0])
			}

		default:
			return fmt.Errorf("readItemNode: unsupported attr type: %s", attr)
		}
	}

	for node := node.ChildNode(); node != nil; node = node.NextNode() {
		if err := m.readItemChildNode(node, parentTile, &item, depth+1); err != nil {
			return fmt.Errorf("error reading tile child node: %v", err)
		}
	}

	if item.otbItemTypeID == 0 {
		glog.Errorf("bad item 0 at tile parent item %v / parent tile %v; skipping", parentItem, parentTile)
		return nil
	}

	if parentItem != nil {
		// TODO: parentItem.AddChild(...)
	} else if parentTile != nil {
		//otbItem := m.things.Temp__GetItemFromOTB(item.GetServerType(), 0)
		//if otbItem.Group == itemsotb.ITEM_GROUP_GROUND {
		//}
		parentTile.addItem(item)
	}

	return nil
}

func (m *Map) readItemChildNode(node *otb.OTBNode, parentTile *mapTile, parentItem *mapItem, depth int) error {
	switch MapNodeType(node.NodeType()) {
	case OTBM_ITEM:
		return m.readItemNode(node, parentTile, parentItem, depth)
	default:
		return fmt.Errorf("readItemChildNode: unsupported node type 0x%02x", node.NodeType())
	}
	return nil
}

func (m *Map) readTownsNode(node *otb.OTBNode) error {
	glog.V(2).Infof("towns")
	for node := node.ChildNode(); node != nil; node = node.NextNode() {
		if err := m.readTownsChildNode(node); err != nil {
			return fmt.Errorf("error reading towns child node: %v", err)
		}
	}
	return nil
}

func (m *Map) readTownsChildNode(node *otb.OTBNode) error {
	switch MapNodeType(node.NodeType()) {
	case OTBM_TOWN:
		return m.readTownNode(node)
	default:
		return fmt.Errorf("readTownNode: unsupported node type 0x%02x", node.NodeType())
	}
	return nil
}

func (m *Map) readTownNode(node *otb.OTBNode) error {
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
		return fmt.Errorf("error reading prop id of town node: %v", err)
	}

	var nameSize uint16
	if err := binary.Read(propBuf, binary.LittleEndian, &nameSize); err != nil {
		return fmt.Errorf("error reading prop name's size in town node: %v", err)
	}

	nameB := make([]byte, nameSize)
	n, err := propBuf.Read(nameB)
	if err != nil {
		return fmt.Errorf("error reading prop name of town node: %v", err)
	}
	if n != int(nameSize) {
		return fmt.Errorf("did not read entire name in town node: got %d, want %d", n, nameSize)
	}
	props.name = string(nameB) // assume utf8, I suppose

	if err := binary.Read(propBuf, binary.LittleEndian, &props.templePos); err != nil {
		return fmt.Errorf("error reading prop templePos of town node: %v", err)
	}

	glog.V(2).Infof(" town %s (%d) with temple at %d,%d,%d", props.name, props.id, props.templePos.TempleX, props.templePos.TempleY, props.templePos.TempleFloor)

	if m.defaultPlayerSpawnPoint == 0 {
		m.defaultPlayerSpawnPoint = posFromCoord(props.templePos.TempleX, props.templePos.TempleY, props.templePos.TempleFloor)
		glog.V(2).Infof("  this town is now the default spawn point %v", m.defaultPlayerSpawnPoint)
	}

	// skipping child nodes

	return nil
}

func (m *Map) readWaypointsNode(node *otb.OTBNode) error {
	glog.V(2).Infof("waypoints")
	for node := node.ChildNode(); node != nil; node = node.NextNode() {
		if err := m.readWaypointsChildNode(node); err != nil {
			return fmt.Errorf("error reading waypoints child node: %v", err)
		}
	}
	return nil
}

func (m *Map) readWaypointsChildNode(node *otb.OTBNode) error {
	switch MapNodeType(node.NodeType()) {
	case OTBM_WAYPOINT:
		return m.readWaypointNode(node)
	default:
		return fmt.Errorf("readWaypointNode: unsupported node type 0x%02x", node.NodeType())
	}
	return nil
}

func (m *Map) readWaypointNode(node *otb.OTBNode) error {
	propBuf := node.PropsBuffer()
	type propType struct {
		name string
		pos  struct {
			X, Y  uint16
			Floor uint8
		}
	}
	props := propType{}

	var nameSize uint16
	if err := binary.Read(propBuf, binary.LittleEndian, &nameSize); err != nil {
		return fmt.Errorf("error reading prop name's size in waypoint node: %v", err)
	}

	nameB := make([]byte, nameSize)
	n, err := propBuf.Read(nameB)
	if err != nil {
		return fmt.Errorf("error reading prop name of waypoint node: %v", err)
	}
	if n != int(nameSize) {
		return fmt.Errorf("did not read entire name in waypoint node: got %d, want %d", n, nameSize)
	}
	props.name = string(nameB) // assume utf8, I suppose

	if err := binary.Read(propBuf, binary.LittleEndian, &props.pos); err != nil {
		return fmt.Errorf("error reading prop pos of waypoint node: %v", err)
	}

	glog.V(2).Infof(" waypoint %s with pos at %d,%d,%d", props.name, props.pos.X, props.pos.Y, props.pos.Floor)

	if m.defaultPlayerSpawnPoint == 0 {
		m.defaultPlayerSpawnPoint = posFromCoord(props.pos.X, props.pos.Y, props.pos.Floor)
		glog.V(2).Infof("  this waypoint is now the default spawn point %v", m.defaultPlayerSpawnPoint)
	}

	// skipping child nodes

	return nil
}
