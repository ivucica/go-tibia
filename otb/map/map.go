// Package otbm implements an OTBM map file format reader and a gameworld map data source.
package otbm

import (
	"badc0de.net/pkg/go-tibia/gameworld"
	"badc0de.net/pkg/go-tibia/otb"
	"github.com/golang/glog"

	"encoding/binary"
	"fmt"
	"io"
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

func posFromCoord(x, y uint16, floor uint8) pos {
	return pos((uint64(floor) << 32) | (uint64(y) << 16) | uint64(x))
}

type Map struct {
	otb.OTB
	gameworld.MapDataSource
	tiles map[pos]mapTile
}

type mapTile struct {
	gameworld.MapTile
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

type rootHeader struct {
	Ver                          uint32
	Width, Height                uint16
	ItemsVerMajor, ItemsVerMinor uint32
}

// New reads an OTB file from a given reader.
func New(r io.ReadSeeker) (*Map, error) {
	f, err := otb.NewOTB(r)
	if err != nil {
		return nil, fmt.Errorf("newotbm failed to use fileloader: %s", err)
	}

	otb := Map{
		OTB: *f,
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
		glog.V(2).Infof("tile area")
		m.readTileAreaNode(node)
	case OTBM_TOWNS:
		glog.V(2).Infof("towns")
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
		return nil, fmt.Errorf("error reading coordinates of tile area node: %v", err)
	}

	glog.V(2).Infof(" at %d,%d,%d", props.X, props.Y, props.Floor)

	for node := m.ChildNode(node); node != nil; node = node.NextNode() {
		if mapData, err := m.readTileAreaChildNode(node); err == nil {
			mapData = mapData
		} else {
			return nil, fmt.Errorf("error reading tile area child node: %v", err)
		}
	}
	return nil, nil
}

func (m *Map) readTileAreaChildNode(node *otb.OTBNode) (MapData, error) { // TODO: this won't return mapdata.
	switch MapNodeType(node.NodeType()) {
	case OTBM_TILE:
		glog.V(2).Infof(" tile")
	default:
		return nil, fmt.Errorf("readTileAreaChildNode: unsupported node type 0x%02x", node.NodeType())
	}
	return nil, nil
}
