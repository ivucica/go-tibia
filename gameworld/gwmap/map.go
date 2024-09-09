package gwmap

import (
	"badc0de.net/pkg/go-tibia/dat"
	tnet "badc0de.net/pkg/go-tibia/net"
)

////// Interfaces //////

// MapDataSource is an interface for a data source that provides map data. It
// can be used to get map tiles, creatures, etc. via locally loaded files, or
// using an RPC call to a remote tile server, etc.
type MapDataSource interface {
	GetMapTile(x, y uint16, floor uint8) (MapTile, error)
	GetCreatureByIDBytes(id [4]byte) (Creature, error)
	GetCreatureByID(CreatureID) (Creature, error)
	AddCreature(creature Creature) error
	RemoveCreatureByID(CreatureID) error
	GetAmbientLight() (ambientColor dat.DatasetColor, ambientLevel uint8) // This might not belong in this interface: it's useful for renderer, but how would we combine multiple backing data sources and the fact this is really gameworld-wide? It might belong in gameworld instead.

	Private_And_Temp__DefaultPlayerSpawnPoint(CreatureID) tnet.Position
}

// MapTile is an interface for a map tile. A map tile is a single tile on the
// map grid. It contains a list of items and creatures that are on that tile.
type MapTile interface {
	GetItem(idx int) (MapItem, error)
	AddCreature(creature Creature) error
	GetCreature(idx int) (Creature, error)
	RemoveCreature(Creature) error
}

// MapItem is an interface for an item on a map tile. An item is anything that
// can be placed on a map tile, such as a tree, a rock, a corpse, etc.
type MapItem interface {
	GetServerType() uint16
	GetCount() uint16
}

// MapTileEventSubscriber is an interface for an object that can subscribe to
// events that occur on a map tile. This is important so the game server can be
// notified either locally or over an RPC call when a creature moves, its health
// is updated, or an item is added or removed or updated, etc.
type MapTileEventSubscriber interface {
}
