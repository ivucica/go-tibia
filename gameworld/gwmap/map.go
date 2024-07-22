package gwmap

import (
	"badc0de.net/pkg/go-tibia/dat"
	tnet "badc0de.net/pkg/go-tibia/net"
)

////// Interfaces //////

type MapDataSource interface {
	GetMapTile(x, y uint16, floor uint8) (MapTile, error)
	GetCreatureByIDBytes(id [4]byte) (Creature, error)
	GetCreatureByID(CreatureID) (Creature, error)
	AddCreature(creature Creature) error
	RemoveCreatureByID(CreatureID) error
	GetAmbientLight() (ambientColor dat.DatasetColor, ambientLevel uint8) // This might not belong in this interface: it's useful for renderer, but how would we combine multiple backing data sources and the fact this is really gameworld-wide? It might belong in gameworld instead.

	Private_And_Temp__DefaultPlayerSpawnPoint(CreatureID) tnet.Position
}

type MapTile interface {
	GetItem(idx int) (MapItem, error)
	AddCreature(creature Creature) error
	GetCreature(idx int) (Creature, error)
	RemoveCreature(Creature) error
}

type MapItem interface {
	GetServerType() uint16
	GetCount() uint16
}

type MapTileEventSubscriber interface {
}
