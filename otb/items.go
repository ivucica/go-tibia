package otb

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"

	"github.com/golang/glog"
)

type ItemsOTB struct {
	OTB
	Version ItemsOTBVersion
	Items   []ItemOTB

	ClientIDToArrayIndex map[uint16]int
	ServerIDToArrayIndex map[uint16]int
}

type (
	ItemsOTBAttribute uint8
	ItemsOTBDataSize  uint16
	ItemsOTBFlags     uint32
)

const (
	ROOT_ATTR_VERSION = 0x01
)

const (
	CLIENT_VERSION_750                     = 1
	CLIENT_VERSION_755                     = 2
	CLIENT_VERSION_760, CLIENT_VERSION_770 = 3, 3
	CLIENT_VERSION_780                     = 4
	CLIENT_VERSION_790                     = 5
	CLIENT_VERSION_792                     = 6
	CLIENT_VERSION_800                     = 7
	CLIENT_VERSION_810                     = 8
	CLIENT_VERSION_811                     = 9
	CLIENT_VERSION_820                     = 10
	CLIENT_VERSION_830                     = 11
	CLIENT_VERSION_840                     = 12
	CLIENT_VERSION_841                     = 13
	CLIENT_VERSION_842                     = 14
	CLIENT_VERSION_850                     = 15
	CLIENT_VERSION_854_BAD                 = 16
	CLIENT_VERSION_854                     = 17
	CLIENT_VERSION_855                     = 18
	CLIENT_VERSION_860_OLD                 = 19
	CLIENT_VERSION_860                     = 20
	CLIENT_VERSION_861                     = 21
	CLIENT_VERSION_862                     = 22
	CLIENT_VERSION_870                     = 23
)

const (
	ITEM_GROUP_NONE = iota
	ITEM_GROUP_GROUND
	ITEM_GROUP_CONTAINER
	ITEM_GROUP_WEAPON     // deprecated
	ITEM_GROUP_AMMUNITION // deprecated
	ITEM_GROUP_ARMOR      // deprecated
	ITEM_GROUP_CHARGES
	ITEM_GROUP_TELEPORT   // deprecated
	ITEM_GROUP_MAGICFIELD // deprecated
	ITEM_GROUP_WRITEABLE  // deprecated
	ITEM_GROUP_KEY        // deprecated
	ITEM_GROUP_SPLASH
	ITEM_GROUP_FLUID
	ITEM_GROUP_DOOR // deprecated
	ITEM_GROUP_DEPRECATED
	ITEM_GROUP_LAST
)

const (
	FLAG_BLOCK_SOLID = 1 << iota
	FLAG_BLOCK_PROJECTILE
	FLAG_BLOCK_PATHFIND
	FLAG_HAS_HEIGHT
	FLAG_USEABLE
	FLAG_PICKUPABLE
	FLAG_MOVEABLE
	FLAG_STACKABLE
	FLAG_FLOORCHANGEDOWN
	FLAG_FLOORCHANGENORTH
	FLAG_FLOORCHANGEEAST
	FLAG_FLOORCHANGESOUTH
	FLAG_FLOORCHANGEWEST
	FLAG_ALWAYSONTOP
	FLAG_READABLE
	FLAG_ROTABLE
	FLAG_HANGABLE
	FLAG_VERTICAL
	FLAG_HORIZONTAL
	FLAG_CANNOTDECAY
	FLAG_ALLOWDISTREAD
	FLAG_UNUSED
	FLAG_CLIENTCHARGES // deprecated
	FLAG_LOOKTHROUGH
)

const (
	ITEM_ATTR_FIRST    = 0x10
	ITEM_ATTR_SERVERID = iota + 0x10 - 1
	ITEM_ATTR_CLIENTID
	ITEM_ATTR_NAME  // deprecated
	ITEM_ATTR_DESCR // deprecated
	ITEM_ATTR_SPEED
	ITEM_ATTR_SLOT         // deprecated
	ITEM_ATTR_MAXITEMS     // deprecated
	ITEM_ATTR_WEIGHT       // deprecated
	ITEM_ATTR_WEAPON       // deprecated
	ITEM_ATTR_AMU          // deprecated
	ITEM_ATTR_ARMOR        // deprecated
	ITEM_ATTR_MAGLEVEL     // deprecated
	ITEM_ATTR_MAGFIELDTYPE // deprecated
	ITEM_ATTR_WRITEABLE    // deprecated
	ITEM_ATTR_ROTATETO     // deprecated
	ITEM_ATTR_DECAY        // deprecated
	ITEM_ATTR_SPRITEHASH
	ITEM_ATTR_MINIMAPCOLOR
	ITEM_ATTR_07
	ITEM_ATTR_08
	ITEM_ATTR_LIGHT

	// 1byte aligned
	ITEM_ATTR_DECAY2     // deprecated
	ITEM_ATTR_WEAPON2    // deprecated
	ITEM_ATTR_AMU2       // deprecated
	ITEM_ATTR_ARMOR2     // deprecated
	ITEM_ATTR_WRITEABLE2 // deprecated
	ITEM_ATTR_LIGHT2
	ITEM_ATTR_TOPORDER
	ITEM_ATTR_WRITEABLE3 // deprecated
	ITEM_ATTR_LAST
)

type itemsOTBRootNodeVersion struct {
	DataSize ItemsOTBDataSize
	Version  ItemsOTBVersion
}
type ItemsOTBVersion struct {
	MajorVersion, MinorVersion, BuildNumber uint32
	CSDVersion                              [128]uint8
}

func stringFromCStr(cstr []byte) string {
	idx := bytes.IndexByte(cstr, 0x00)
	if idx == -1 {
		idx = len(cstr)
	}
	return string(cstr[:idx])
}

func NewItemsOTB(r io.ReadSeeker) (*ItemsOTB, error) {
	f, err := NewOTB(r)
	if err != nil {
		return nil, fmt.Errorf("newitemsotb failed to use fileloader: %s", err)
	}

	otb := ItemsOTB{
		OTB:                  *f,
		ClientIDToArrayIndex: make(map[uint16]int),
		ServerIDToArrayIndex: make(map[uint16]int),
	}

	root := otb.ChildNode(nil)
	if root == nil {
		return nil, fmt.Errorf("nil root node")
	}

	props := root.PropsBuffer()
	var flags uint32
	if err := binary.Read(props, binary.LittleEndian, &flags); err != nil {
		return nil, fmt.Errorf("error reading itemsotb root node flags: %v", err)
	}
	flags = flags // seemingly unused

	var attr ItemsOTBAttribute
	if err := binary.Read(props, binary.LittleEndian, &attr); err != nil {
		return nil, fmt.Errorf("error reading itemsotb root node attr: %v", err)
	}
	switch attr {
	case ROOT_ATTR_VERSION:
		var vers itemsOTBRootNodeVersion
		if err := binary.Read(props, binary.LittleEndian, &vers); err != nil {
			return nil, fmt.Errorf("error reading itemsotb root node attr 'version': %v", err)
		}
		if vers.DataSize != /* sizeof ItemsOTBRootNodeVersion */ 4+4+4+128 {
			return nil, fmt.Errorf("bad size of itemsotb root node attr 'version': %v", vers.DataSize)
		}

		glog.V(2).Infof("items.otb version %d.%d.%d, csd %s", vers.Version.MajorVersion, vers.Version.MinorVersion, vers.Version.BuildNumber, stringFromCStr(vers.Version.CSDVersion[:]))
		if vers.Version.MajorVersion == 0xFFFFFFFF {
			glog.Warning("generic items.otb found, skipping version check")
		} else {
			if vers.Version.MajorVersion != 3 {
				return nil, fmt.Errorf("unsupported itemsotb major version: got %d, want %d", vers.Version.MajorVersion, 3)
			}

			minVersion := CLIENT_VERSION_854 // development dat files are 8.54
			maxVersion := CLIENT_VERSION_870 // reference source code was 8.70
			if vers.Version.MinorVersion < uint32(minVersion) || vers.Version.MinorVersion > uint32(maxVersion) {
				return nil, fmt.Errorf("unsupported itemsotb major version: got %d, want [%d, %d]", vers.Version.MinorVersion, minVersion, maxVersion)
			}
		}
		otb.Version = vers.Version
	default:
		// ignore, apparently
	}

	if otb.ChildNode(root) == nil {
		return nil, fmt.Errorf("no children in root node")
	}

	for node := otb.ChildNode(root); node != nil; node = node.NextNode() {
		if item, err := otb.readChildNode(node); err == nil {
			if id, ok := item.Attributes[ITEM_ATTR_CLIENTID]; ok {
				id := id.(uint16)
				otb.ClientIDToArrayIndex[id] = len(otb.Items)
			}
			if id, ok := item.Attributes[ITEM_ATTR_SERVERID]; ok {
				id := id.(uint16)
				otb.ServerIDToArrayIndex[id] = len(otb.Items)
			}
			// TODO(ivucica): main OTB loader could give us a count of child nodes, and we could use that to preallocate space instead of appending all the time
			otb.Items = append(otb.Items, *item)
		} else {
			return nil, err
		}
	}
	return &otb, nil
}

func (*ItemsOTB) readChildNode(node *OTBNode) (*ItemOTB, error) {
	props := node.PropsBuffer()

	var flags ItemsOTBFlags
	if err := binary.Read(props, binary.LittleEndian, &flags); err != nil {
		return nil, fmt.Errorf("error reading itemsotb child node flags: %v", err)
	}

	item := ItemOTB{
		Group:      int(node.NodeType()),
		Flags:      flags,
		Attributes: make(map[ItemsOTBAttribute]interface{}),
	}

	var attr ItemsOTBAttribute
	for err := binary.Read(props, binary.LittleEndian, &attr); err == nil; err = binary.Read(props, binary.LittleEndian, &attr) {
		var datalen ItemsOTBDataSize
		if err := binary.Read(props, binary.LittleEndian, &datalen); err != nil {
			return nil, fmt.Errorf("error reading itemsotb child node data len: %v", err)
		}
		switch attr {
		case ITEM_ATTR_SERVERID: // TODO: max is 20000
			fallthrough
		case ITEM_ATTR_CLIENTID:
			fallthrough
		case ITEM_ATTR_SPEED:
			var val uint16
			if datalen != 2 {
				return nil, fmt.Errorf("invalid attribute %d size: got %d, want %d", attr, datalen, 2)
			}
			if err := binary.Read(props, binary.LittleEndian, &val); err != nil {
				return nil, fmt.Errorf("error reading itemsotb child node 2b attribute %d: %v", attr, err)
			}
			item.Attributes[attr] = val
		case ITEM_ATTR_TOPORDER:
			var val uint8
			if datalen != 1 {
				return nil, fmt.Errorf("invalid attribute %d size: got %d, want %d", attr, datalen, 1)
			}
			if err := binary.Read(props, binary.LittleEndian, &val); err != nil {
				return nil, fmt.Errorf("error reading itemsotb child node 1b attribute %d: %v", attr, err)
			}
			item.Attributes[attr] = val
		case ITEM_ATTR_LIGHT2:
			if datalen != 4 {
				return nil, fmt.Errorf("invalid attribute %d size: got %d, want %d", attr, datalen, 4)
			}
			var val ItemOTBLight
			if err := binary.Read(props, binary.LittleEndian, &val); err != nil {
				return nil, fmt.Errorf("error reading itemsotb child node light attribute %d: %v", attr, err)
			}
			item.Attributes[attr] = val
		default:
			// we could get bytes but ignore the value (which means 'skip' if you squint).
			// however let's pretend it's useful to store them in the map
			item.Attributes[attr] = props.Next(int(datalen))
		}
	}
	return &item, nil
}

func (otb *ItemsOTB) ItemByServerID(serverID uint16) (*ItemOTB, error) {
	if idx, ok := otb.ServerIDToArrayIndex[serverID]; ok {
		return &otb.Items[idx], nil
	} else {
		return nil, fmt.Errorf("item not found with server id: %d", serverID)
	}
}

func (otb *ItemsOTB) ItemByClientID(clientID uint16) (*ItemOTB, error) {
	if idx, ok := otb.ClientIDToArrayIndex[clientID]; ok {
		return &otb.Items[idx], nil
	} else {
		return nil, fmt.Errorf("item not found with client id: %d", clientID)
	}
}

type ItemOTB struct {
	Group      int // enum type
	Flags      ItemsOTBFlags
	Attributes map[ItemsOTBAttribute]interface{}
}

type ItemOTBLight struct {
	LightLevel uint16
	LightColor uint16
}
