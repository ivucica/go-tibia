package otbm

import (
	"fmt"
)

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
