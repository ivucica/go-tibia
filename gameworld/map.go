package gameworld

import (
	tnet "badc0de.net/pkg/go-tibia/net"
	"badc0de.net/pkg/go-tibia/otb/items"
	"encoding/binary"
	"fmt"

	"github.com/golang/glog"
)

var (
	ItemNotFound     error
	CreatureNotFound error
)

func init() {
	ItemNotFound = fmt.Errorf("item not found")
	CreatureNotFound = fmt.Errorf("creature not found")
}

////// Interfaces //////

type MapDataSource interface {
	GetMapTile(x, y uint16, floor uint8) (MapTile, error)
	GetCreatureByIDBytes(id [4]byte) (Creature, error)
	GetCreatureByID(CreatureID) (Creature, error)
	AddCreature(creature Creature) error
	RemoveCreatureByID(CreatureID) error

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
}

type MapTileEventSubscriber interface {
}

////////////////////////

func (c *GameworldConnection) floorDescription(outMap *tnet.Message, x, y uint16, z uint8, width, height uint16) error {
	var skip int
	for nx := x; nx < x+width; nx++ {
		for ny := y; ny < y+height; ny++ {
			tile, err := c.server.mapDataSource.GetMapTile(nx, ny, z)
			if err != nil {
				return fmt.Errorf("failed to get tile %d %d %d: %v", nx, ny, z, err)
			}
			skip, err = c.tileDescription(outMap, tile, skip)
			if err != nil {
				return fmt.Errorf("failed to send tile desc for %d %d %d: %v", nx, ny, z, err)
			}
		}
	}
	if skip > 0 {
		// little endian of 0xFF00 & skiptiles
		outMap.Write([]byte{byte(skip), 0xFF})
		skip = 0
	}

	return nil
}

func (c *GameworldConnection) tileDescription(outMap *tnet.Message, tile MapTile, skip int) (int, error) {
	emptyTile := func() (int, error) {
		if skip >= 0xFF {
			// little endian of 0xFF00 & skiptiles
			outMap.Write([]byte{0xFF, 0xFF})
			skip -= 0xFF
		} else {
			skip++
		}
		return skip, nil
	}

	ground, err := tile.GetItem(0)
	if err != nil {
		if err == ItemNotFound {
			return emptyTile()
		}

		return skip, err
	}
	if ground == nil {
		glog.Warningf("Bug in map data source: returned item is nil, but error is not ItemNotFound")
		return emptyTile()
	}
	groundOTBItem := c.server.things.Temp__GetItemFromOTB(ground.GetServerType(), c.clientVersion)
	if groundOTBItem.Group != itemsotb.ITEM_GROUP_GROUND {
		// TODO(ivucica): support tiles with only non-ground items or with only creatures (although, does that make sense?)
		return emptyTile()
	}

	groundClientID := c.server.things.Temp__GetClientIDForServerID(ground.GetServerType(), c.clientVersion)
	if groundClientID == 0 {
		// some error getting client ID
		return emptyTile()
	}

	// little endian of 0xFF00 & skiptiles
	outMap.Write([]byte{byte(skip), 0xFF})
	skip = 0
	outMap.Write([]byte{
		byte(groundClientID % 256), byte(groundClientID / 256), // ground
	})

	if groundOTBItem.Flags&itemsotb.FLAG_STACKABLE != 0 || groundOTBItem.Group == itemsotb.ITEM_GROUP_FLUID || groundOTBItem.Group == itemsotb.ITEM_GROUP_SPLASH {
		// either count or fluid color
		outMap.Write([]byte{
			byte(1),
		})
	}

	// add any creatures on this tile
	for idx := 0; ; idx++ {
		if cr, err := tile.GetCreature(idx); err != CreatureNotFound {
			if err != nil {
				return skip, err
			}
			if err := c.creatureDescription(outMap, cr); err != nil {
				return skip, err
			}
		} else {
			// We know err == CreatureNotFound or nil
			break
		}
	}

	return skip, nil
}

func (c *GameworldConnection) initialAppearMap(outMap *tnet.Message) error {
	outMap.Write([]byte{0x64}) // full map desc

	playerID, err := c.PlayerID()
	if err != nil {
		return err
	}

	creature, err := c.server.mapDataSource.GetCreatureByID(playerID)
	if err != nil {
		return err
	}

	pos := creature.GetPos()
	outMap.WriteTibiaPosition(pos)

	glog.V(2).Infof("initialAppearMap for player %d at %d %d %d", playerID, pos.X, pos.Y, pos.Floor)

	if pos.Floor != 7 {
		return fmt.Errorf("TEMPORARILY unsupported initial location. Floor currently must be 7.")
	}

	for floor := 7; floor >= 0; floor-- {
		glog.V(2).Infof("sending floor %d", floor)
		if err := c.floorDescription(outMap, pos.X+uint16(7-floor-(18/2-1)), pos.Y+uint16(7-floor-(14/2-1)), uint8(floor), 18, 14); err != nil {
			return fmt.Errorf("failed to send floor %d during initialAppearMap: %v", floor, err)
		}
	}
	glog.V(2).Infof("initial map sent")

	return nil
}

func (c *GameworldConnection) creatureDescription(outMap *tnet.Message, cr Creature) error {
	// TODO(ivucica): support not sending the whole creature (i.e. support something other than 0x61)
	outMap.Write([]byte{
		0x61, 0x00, // not known creature thingid
		0x00, 0x00, 0x00, 0x00, // remove
	})

	// 0xAA, 0xBB, 0xCC, 0xDD, // creature id (this is, currently, player's id)
	if err := binary.Write(outMap, binary.LittleEndian, cr.GetID()); err != nil {
		return err
	}

	outMap.WriteTibiaString(cr.GetName())

	outMap.Write([]byte{
		100,             // health
		0,               // dir,
		128, 0, 5, 2, 0, // outfit
		0, 0, // looktype ex u16
		0, 0, // light level and color
		100, 0, // step speed
		0, //skull
		0, // party shield
		0, // 0x61, therefore send war emblem
		0, // player can walk through
	})

	return nil
}
