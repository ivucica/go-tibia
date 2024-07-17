package itemsotb

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"strings"

	"badc0de.net/pkg/go-tibia/otb"
	"github.com/golang/glog"
)

type Items struct {
	otb.OTB
	Version ItemsVersion
	Items   []Item

	ClientIDToArrayIndex map[uint16]int
	ServerIDToArrayIndex map[uint16]int

	ExtantClientItemIDs       []uint16
	ExtantServerItemIDs       []uint16
	ExtantClientItemArrayIdxs []int
	ExtantServerItemArrayIdxs []int

	ServerIDToExtantClientItemArrayIDXs map[uint16]int // a strange hack allowing us to seek within ExtantClientItemArrayIdxs based on server ID

	MinClientID, MaxClientID uint16
	MinServerID, MaxServerID uint16
}

type (
	ItemsAttribute uint8
	ItemsDataSize  uint16
	ItemsFlags     uint32
)

const (
	ROOT_ATTR_VERSION = 0x01
)

// Enumeration containing recognized protocol versions for which a particular
// items.otb file might be targeted.
//
// Implementation detail: iota is not used primarily for easier referencing in
// case of an error.
const (
	CLIENT_VERSION_750                     = ClientVersion(1)
	CLIENT_VERSION_755                     = ClientVersion(2)
	CLIENT_VERSION_760, CLIENT_VERSION_770 = ClientVersion(3), ClientVersion(3)
	CLIENT_VERSION_780                     = ClientVersion(4)
	CLIENT_VERSION_790                     = ClientVersion(5)
	CLIENT_VERSION_792                     = ClientVersion(6)
	CLIENT_VERSION_800                     = ClientVersion(7)
	CLIENT_VERSION_810                     = ClientVersion(8)
	CLIENT_VERSION_811                     = ClientVersion(9)
	CLIENT_VERSION_820                     = ClientVersion(10)
	CLIENT_VERSION_830                     = ClientVersion(11)
	CLIENT_VERSION_840                     = ClientVersion(12)
	CLIENT_VERSION_841                     = ClientVersion(13)
	CLIENT_VERSION_842                     = ClientVersion(14)
	CLIENT_VERSION_850                     = ClientVersion(15)
	CLIENT_VERSION_854_BAD                 = ClientVersion(16)
	CLIENT_VERSION_854                     = ClientVersion(17)
	CLIENT_VERSION_855                     = ClientVersion(18)
	CLIENT_VERSION_860_OLD                 = ClientVersion(19)
	CLIENT_VERSION_860                     = ClientVersion(20)
	CLIENT_VERSION_861                     = ClientVersion(21)
	CLIENT_VERSION_862                     = ClientVersion(22)
	CLIENT_VERSION_870                     = ClientVersion(23)
)

// Enumeration containing recognized protocol versions for which a particular
// items.otb file might be targeted.
type ClientVersion uint32

// String implements the stringer interface.
func (v ClientVersion) String() string {
	switch v {
	case CLIENT_VERSION_750:
		return "7.50"
	case CLIENT_VERSION_755:
		return "7.55"
	case CLIENT_VERSION_760:
		return "7.60 / 7.70"
	case CLIENT_VERSION_780:
		return "7.80"
	case CLIENT_VERSION_790:
		return "7.90"
	case CLIENT_VERSION_792:
		return "7.92"
	case CLIENT_VERSION_800:
		return "8.00"
	case CLIENT_VERSION_810:
		return "8.10"
	case CLIENT_VERSION_811:
		return "8.11"
	case CLIENT_VERSION_820:
		return "8.20"
	case CLIENT_VERSION_830:
		return "8.30"
	case CLIENT_VERSION_840:
		return "8.40"
	case CLIENT_VERSION_841:
		return "8.41"
	case CLIENT_VERSION_842:
		return "8.42"
	case CLIENT_VERSION_850:
		return "8.50"
	case CLIENT_VERSION_854_BAD:
		return "8.54 (bad)"
	case CLIENT_VERSION_854:
		return "8.54"
	case CLIENT_VERSION_855:
		return "8.55"
	case CLIENT_VERSION_860_OLD:
		return "8.60 (old)"
	case CLIENT_VERSION_860:
		return "8.60"
	case CLIENT_VERSION_861:
		return "8.61"
	case CLIENT_VERSION_862:
		return "8.62"
	case CLIENT_VERSION_870:
		return "8.70"
	}
	return fmt.Sprintf("client version %d unknown", v)
}

// Enumeration containing which overarching item group this item belongs to.
//
// Useful primarily for editors.
type ItemGroup int

const (
	ITEM_GROUP_NONE ItemGroup = iota
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

func (g ItemGroup) String() string {
	switch g {
	case ITEM_GROUP_NONE:
		return "none"
	case ITEM_GROUP_GROUND:
		return "ground"
	case ITEM_GROUP_CONTAINER:
		return "container"
	case ITEM_GROUP_WEAPON:
		return "weapon"
	case ITEM_GROUP_AMMUNITION:
		return "ammunition"
	case ITEM_GROUP_ARMOR:
		return "armor"
	case ITEM_GROUP_CHARGES:
		return "charges"
	case ITEM_GROUP_TELEPORT:
		return "teleport"
	case ITEM_GROUP_MAGICFIELD:
		return "magic field"
	case ITEM_GROUP_WRITEABLE:
		return "writeable"
	case ITEM_GROUP_KEY:
		return "key"
	case ITEM_GROUP_SPLASH:
		return "splash"
	case ITEM_GROUP_FLUID:
		return "fluid"
	case ITEM_GROUP_DOOR:
		return "door"
	case ITEM_GROUP_DEPRECATED:
		return "deprecated"
	case ITEM_GROUP_LAST:
		return "last (invalid value)"
	default:
		return "invalid item group"
	}
}

// Enumeration containing possible bits in the `flags` bitmask of an item.
const (
	FLAG_BLOCK_SOLID ItemsFlags = 1 << iota
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
	FLAG_ANIMATION
	FLAG_WALKSTACK

	FLAG_LAST
)

func (f ItemsFlags) String() string {
	out := make([]string, 0, 32)
	for bit := FLAG_BLOCK_SOLID; bit < FLAG_LAST; bit <<= 1 {
		if f&bit == 0 {
			continue
		}
		var desc string
		switch bit {
		case FLAG_BLOCK_SOLID:
			desc = "block solid"
		case FLAG_BLOCK_PROJECTILE:
			desc = "block projectile"
		case FLAG_BLOCK_PATHFIND:
			desc = "block pathfind"
		case FLAG_HAS_HEIGHT:
			desc = "has height"
		case FLAG_USEABLE:
			desc = "useable"
		case FLAG_PICKUPABLE:
			desc = "pickupable"
		case FLAG_MOVEABLE:
			desc = "moveable"
		case FLAG_FLOORCHANGEDOWN:
			desc = "floor change down"
		case FLAG_FLOORCHANGENORTH:
			desc = "floor change north"
		case FLAG_FLOORCHANGEEAST:
			desc = "floor change east"
		case FLAG_FLOORCHANGESOUTH:
			desc = "floor change south"
		case FLAG_FLOORCHANGEWEST:
			desc = "floor change west"
		case FLAG_ALWAYSONTOP:
			desc = "always on top"
		case FLAG_READABLE:
			desc = "readable"
		case FLAG_ROTABLE:
			desc = "rotable"
		case FLAG_HANGABLE:
			desc = "hangable"
		case FLAG_VERTICAL:
			desc = "vertical"
		case FLAG_HORIZONTAL:
			desc = "horizontal"
		case FLAG_CANNOTDECAY:
			desc = "cannot decay"
		case FLAG_ALLOWDISTREAD:
			desc = "allow dist read"
		case FLAG_UNUSED:
			desc = "unused"
		case FLAG_CLIENTCHARGES:
			desc = "client charges"
		case FLAG_LOOKTHROUGH:
			desc = "lookthrough"
		case FLAG_ANIMATION:
			desc = "animation"
		case FLAG_WALKSTACK:
			desc = "walk stack"
		}
		if desc != "" {
			out = append(out, desc)
		}
	}
	return strings.Join(out, ", ")
}

// Enumeration containing recognized attributes in the items.otb file.
const (
	ITEM_ATTR_FIRST    ItemsAttribute = 0x10
	ITEM_ATTR_SERVERID ItemsAttribute = iota + 0x10 - 1
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

func (a ItemsAttribute) String() string {
	switch a {
	case ITEM_ATTR_SERVERID:
		return "server id"
	case ITEM_ATTR_CLIENTID:
		return "client id"
	case ITEM_ATTR_NAME:
		return "name"
	case ITEM_ATTR_DESCR:
		return "description"
	case ITEM_ATTR_SPEED:
		return "speed"
	case ITEM_ATTR_SLOT:
		return "slot"
	case ITEM_ATTR_MAXITEMS:
		return "max items"
	case ITEM_ATTR_WEIGHT:
		return "weight"
	case ITEM_ATTR_WEAPON:
		return "weapon"
	case ITEM_ATTR_AMU:
		return "amu"
	case ITEM_ATTR_ARMOR:
		return "armor"
	case ITEM_ATTR_MAGLEVEL:
		return "magic level"
	case ITEM_ATTR_MAGFIELDTYPE:
		return "magic field type"
	case ITEM_ATTR_WRITEABLE:
		return "writeable"
	case ITEM_ATTR_ROTATETO:
		return "rotate to"
	case ITEM_ATTR_DECAY:
		return "decay"
	case ITEM_ATTR_SPRITEHASH:
		return "spritehash"
	case ITEM_ATTR_MINIMAPCOLOR:
		return "minimap color"
	case ITEM_ATTR_07:
		return "attr 07"
	case ITEM_ATTR_08:
		return "attr 08"
	case ITEM_ATTR_LIGHT:
		return "light"

	// 1byte aligned
	case ITEM_ATTR_DECAY2:
		return "decay2"
	case ITEM_ATTR_WEAPON2:
		return "weapon2"
	case ITEM_ATTR_AMU2:
		return "amu2"
	case ITEM_ATTR_ARMOR2:
		return "armor2"
	case ITEM_ATTR_WRITEABLE2:
		return "writeable2"
	case ITEM_ATTR_LIGHT2:
		return "light2"
	case ITEM_ATTR_TOPORDER:
		return "toporder"
	case ITEM_ATTR_WRITEABLE3:
		return "writeable3"
	case ITEM_ATTR_LAST:
		return "last (invalid value)"
	default:
		return "invalid attribute"
	}
}

type rootNodeVersion struct {
	DataSize ItemsDataSize
	Version  ItemsVersion
}

// ItemsVersion represents the version of the items.otb file.
//
// MajorVersion means a revision of the file format, MinorVersion means the
// targeted protocol version, BuildNumber is an arbitrary number representing
// ther revision of the file, and CSDVersion is a byte array with a C-style
// null-terminated string.
type ItemsVersion struct {
	MajorVersion uint32
	MinorVersion ClientVersion // uint32
	BuildNumber  uint32
	CSDVersion   [128]uint8
}

// CSDVersionAsString formats null-terminated C-style string `CSDArray` from a
// byte array into usual Go string.
func (v ItemsVersion) CSDVersionAsString() string {
	return stringFromCStr(v.CSDVersion[:])
}

// stringFromCStr turns a byte slice representing a null-terminated C-style
// string into a Go string.
func stringFromCStr(cstr []byte) string {
	idx := bytes.IndexByte(cstr, 0x00)
	if idx == -1 {
		idx = len(cstr)
	}
	return string(cstr[:idx])
}

// New reads an OTB file from a given reader.
func New(r io.ReadSeeker) (*Items, error) {
	f, err := otb.NewOTB(r)
	if err != nil {
		return nil, fmt.Errorf("newitemsotb failed to use fileloader: %s", err)
	}

	otb := Items{
		OTB:                  *f,
		ClientIDToArrayIndex: make(map[uint16]int),
		ServerIDToArrayIndex: make(map[uint16]int),
		MinClientID:          0xFFFF, // largest 16bit int; we reduce it below
		MinServerID:          19999,  // 20000 is where other descriptions may begin, like fluids

		ServerIDToExtantClientItemArrayIDXs: make(map[uint16]int),
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

	var attr ItemsAttribute
	if err := binary.Read(props, binary.LittleEndian, &attr); err != nil {
		return nil, fmt.Errorf("error reading itemsotb root node attr: %v", err)
	}
	switch attr {
	case ROOT_ATTR_VERSION:
		var vers rootNodeVersion
		if err := binary.Read(props, binary.LittleEndian, &vers); err != nil {
			return nil, fmt.Errorf("error reading itemsotb root node attr 'version': %v", err)
		}
		if vers.DataSize != /* sizeof ItemsRootNodeVersion */ 4+4+4+128 {
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
			if vers.Version.MinorVersion < minVersion || vers.Version.MinorVersion > maxVersion {
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
				if id < otb.MinClientID {
					otb.MinClientID = id
				}
				if id > otb.MaxClientID {
					otb.MaxClientID = id
				}
				otb.ExtantClientItemIDs = append(otb.ExtantClientItemIDs, id)
				otb.ExtantClientItemArrayIdxs = append(otb.ExtantClientItemArrayIdxs, len(otb.Items))
			}
			if id, ok := item.Attributes[ITEM_ATTR_SERVERID]; ok {
				id := id.(uint16)
				otb.ServerIDToArrayIndex[id] = len(otb.Items)
				if id < otb.MinServerID {
					otb.MinServerID = id
				}
				if id > otb.MaxServerID {
					otb.MaxServerID = id
				}
				otb.ExtantServerItemIDs = append(otb.ExtantServerItemIDs, id)
				otb.ExtantServerItemArrayIdxs = append(otb.ExtantServerItemArrayIdxs, len(otb.Items))
				if _, ok := item.Attributes[ITEM_ATTR_CLIENTID]; ok {
					otb.ServerIDToExtantClientItemArrayIDXs[id] = len(otb.ExtantClientItemArrayIdxs) - 1
				}
				// TODO(ivucica): we should detect duplicate server IDs (duplicate client IDs are, theoretically, permissible)
			}
			// TODO(ivucica): main OTB loader could give us a count of child nodes, and we could use that to preallocate space instead of appending all the time
			otb.Items = append(otb.Items, *item)
		} else {
			return nil, err
		}
	}
	return &otb, nil
}

// readChildNode reads a single "OTB node", as read from an OTB file.
func (*Items) readChildNode(node *otb.OTBNode) (*Item, error) {
	props := node.PropsBuffer()

	var flags ItemsFlags
	if err := binary.Read(props, binary.LittleEndian, &flags); err != nil {
		return nil, fmt.Errorf("error reading itemsotb child node flags: %v", err)
	}

	item := Item{
		Group:      ItemGroup(node.NodeType()),
		Flags:      flags,
		Attributes: make(map[ItemsAttribute]interface{}),
	}

	var attr ItemsAttribute
	for err := binary.Read(props, binary.LittleEndian, &attr); err == nil; err = binary.Read(props, binary.LittleEndian, &attr) {
		var datalen ItemsDataSize
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
			var val Light
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

// ItemByServerID allows lookup of an item stored in an items.otb file based on
// its persistent 'server' ID, which stays fixed between versions, and is used
// by the server-side data storage, by map files, etc.
func (otb *Items) ItemByServerID(serverID uint16) (*Item, error) {
	if idx, ok := otb.ServerIDToArrayIndex[serverID]; ok {
		return &otb.Items[idx], nil
	} else {
		return nil, fmt.Errorf("item not found with server id: %d", serverID)
	}
}

// ItemByClientID allows lookup of an item stored in an items.otb file based on
// its ID used by the network protocol and associated data files.
func (otb *Items) ItemByClientID(clientID uint16) (*Item, error) {
	if idx, ok := otb.ClientIDToArrayIndex[clientID]; ok {
		return &otb.Items[idx], nil
	} else {
		return nil, fmt.Errorf("item not found with client id: %d", clientID)
	}
}

// Item represents a single item stored in the items.otb file.
type Item struct {
	Group      ItemGroup
	Flags      ItemsFlags
	Attributes map[ItemsAttribute]interface{}

	// TODO(ivucica): Consider making XML data public or merging it into OTB data.
	xml *xmlItem
}

// Name returns the name of the item. This may be sourced from XML, if loaded.
func (i *Item) Name() string {
	if i.xml != nil {
		return i.xml.Name
	}
	if name, ok := i.Attributes[ITEM_ATTR_NAME]; ok {
		return name.(string)
	}
	return "unnamed item"
}

// Article returns the name of the item. This will only be sourced from XML, if
// loaded.
//
// If empty, no article should be used; otherwise, in singular, prefix with
// article and a space.
func (i *Item) Article() string {
	if i.xml != nil {
		return i.xml.Article
	}
	return ""
}

// Description returns the description of the item. This may be sourced from XML, if loaded.
//
// If multiple descriptions are supplied, only the first one will be used.
func (i *Item) Description() string {
	if i.xml != nil && len(i.xml.Attributes["description"]) > 0 {
		return i.xml.Attributes["description"][0]
	}
	if name, ok := i.Attributes[ITEM_ATTR_NAME]; ok {
		return name.(string)
	}
	return ""
}

// ClientID returns the item client ID for the client version for which this OTB
// is intended. If the item does not exist in this client version, zero is
// returned.
func (i *Item) ClientID() uint16 {
	id, ok := i.Attributes[ITEM_ATTR_CLIENTID]
	if !ok {
		return 0
	}
	return id.(uint16)
}

// ServerID returns the item server ID. If the item does not have a server ID
// (which would be highly irregular for an item that appears in the otb file),
// zero is returned.
func (i *Item) ServerID() uint16 {
	id, ok := i.Attributes[ITEM_ATTR_SERVERID]
	if !ok {
		return 0
	}
	return id.(uint16)
}

// Light represents the data structure describing a lit-up item's light attribute
// ITEM_ATTR_LIGHT2, as stored in an items.otb file.
type Light struct {
	LightLevel uint16
	LightColor uint16
}
