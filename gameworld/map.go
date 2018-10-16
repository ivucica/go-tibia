package gameworld

import (
	tnet "badc0de.net/pkg/go-tibia/net"
	"bytes"
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
}

type MapTile interface {
	GetItem(idx int) (MapItem, error)
	AddCreature(creature Creature) error
	GetCreature(idx int) (Creature, error)
	RemoveCreature(Creature) error
}

type MapItem interface {
	GetType() int
}

/////// Concrete implementations ///////

func NewMapDataSource() MapDataSource {
	return &mapDataSource{
		creatures:         map[CreatureID]Creature{},
		generatedMapTiles: map[tnet.Position]MapTile{},
	}
}

type mapDataSource struct {
	creatures map[CreatureID]Creature

	generatedMapTiles map[tnet.Position]MapTile
}
type mapTile struct {
	ground    MapItem
	creatures []Creature
}

type mapItem int

func mapItemOfType(t int) MapItem {
	mi := mapItem(t)
	return &mi
}

func (ds *mapDataSource) GetMapTile(x, y uint16, z uint8) (MapTile, error) {
	if t, ok := ds.generatedMapTiles[tnet.Position{x, y, z}]; ok {
		return t, nil
	}
	generatedMapTile, err := ds.generateMapTile(x, y, z)
	if err != nil {
		return nil, err
	}
	ds.generatedMapTiles[tnet.Position{x, y, z}] = generatedMapTile
	return generatedMapTile, nil

}
func (*mapDataSource) generateMapTile(x, y uint16, z uint8) (MapTile, error) {
	if z == 7 {
		if y == 32768+14/2 {
			switch x % 2 {
			case 0:
				return &mapTile{ground: mapItemOfType(104)}, nil
			case 1:
				return &mapTile{ground: mapItemOfType(103)}, nil
			}
		}
		switch ((y + 3) / 2) % 4 {
		case 0:
			return &mapTile{ground: mapItemOfType(103)}, nil
		case 1:
			return &mapTile{ground: mapItemOfType(104)}, nil
		case 2:
			return &mapTile{ground: mapItemOfType(101)}, nil
		case 3:
			return &mapTile{ground: mapItemOfType(100)}, nil
		default:
			return &mapTile{}, nil
		}
	} else {
		if z == 6 && x == 32768+(18/2)-4 && y == 32768+(14/2) {
			glog.Infof("sending 104 at %d %d %d", x, y, z)
			return &mapTile{ground: mapItemOfType(103)}, nil
		}
		return &mapTile{}, nil
	}
}
func (ds *mapDataSource) GetCreatureByIDBytes(idBytes [4]byte) (Creature, error) {
	buf := bytes.NewBuffer(idBytes[:])
	var id CreatureID
	err := binary.Read(buf, binary.LittleEndian, &id)
	if err != nil {
		return nil, fmt.Errorf("could not decode creature ID from bytes: %v", err)
	}

	return ds.GetCreatureByID(id)
}
func (ds *mapDataSource) GetCreatureByID(id CreatureID) (Creature, error) {
	if creature, ok := ds.creatures[id]; ok {
		return creature, nil
	}
	return nil, CreatureNotFound
}
func (ds *mapDataSource) AddCreature(c Creature) error {
	ds.creatures[c.GetID()] = c
	if t, err := ds.GetMapTile(c.GetPos().X, c.GetPos().Y, c.GetPos().Floor); err != nil {
		return err
	} else {
		glog.Infof("adding creature to %d %d %d", c.GetPos().X, c.GetPos().Y, c.GetPos().Floor)
		return t.AddCreature(c)
	}
}
func (ds *mapDataSource) RemoveCreatureByID(id CreatureID) error {
	c, err := ds.GetCreatureByID(id)
	if err != nil {
		if err == CreatureNotFound {
			return nil
		}
	}

	delete(ds.creatures, id)

	if t, err := ds.GetMapTile(c.GetPos().X, c.GetPos().Y, c.GetPos().Floor); err != nil {
		return err
	} else {
		glog.Infof("deleting creature from %d %d %d", c.GetPos().X, c.GetPos().Y, c.GetPos().Floor)
		return t.RemoveCreature(c)
	}
}

func (t *mapTile) GetCreature(idx int) (Creature, error) {
	if idx >= len(t.creatures) {
		glog.Infof("creature not found; requested idx %d with len %d", idx, len(t.creatures))
		return nil, CreatureNotFound
	}
	return t.creatures[idx], nil
}
func (t *mapTile) GetItem(idx int) (MapItem, error) {
	if idx == 0 && t.ground != nil && t.ground.GetType() != 0 {
		return t.ground, nil
	}
	return nil, ItemNotFound
}
func (t *mapTile) AddCreature(c Creature) error {
	t.creatures = append(t.creatures, c)
	return nil
}

func (t *mapTile) RemoveCreature(cr Creature) error {
	// not t.creatures - 1, in case the creature is not in fact stored on the tile.
	newCs := make([]Creature, 0, len(t.creatures))
	seen := false
	for _, c := range t.creatures {
		if c.GetID() != cr.GetID() {
			seen = true
			newCs = append(newCs, c)
		}
	}
	if !seen {
		glog.Warningf("removing creature %d from tile %d %d %d where it's actually not present", cr.GetID(), cr.GetPos().X, cr.GetPos().Y, cr.GetPos().Floor)
	}
	return nil
}

func (i *mapItem) GetType() int {
	return int(*i)
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
			ground, err := tile.GetItem(0)
			// TODO(ivucica): support tiles with only non-ground items or creatures (although, does that make sense?)
			if ground == nil {
				if skip >= 0xFF {
					outMap.Write([]byte{0xFF, 0xFF})
					skip -= 0xFF
				} else {
					skip++
				}
				continue
			}
			if skip > 0 {
				outMap.Write([]byte{byte(skip), 0xFF})
				skip = 0
			}

			outMap.Write([]byte{
				byte(ground.GetType() % 256), byte(ground.GetType() / 256), // ground
			})

			// add any creatures on this tile
			for idx := 0; ; idx++ {
				if cr, err := tile.GetCreature(idx); err != CreatureNotFound {
					if err != nil {
						return err
					}
					glog.Infof("sending creature (%d %d %d) at idx %d", nx, ny, z, idx)
					if err := c.creatureDescription(outMap, cr); err != nil {
						return err
					}
				} else {
					// err == CreatureNotFound
					glog.Infof("done with creatures (%d %d %d) at idx %d", nx, ny, z, idx)
					break
				}
			}

			// mark tile as done.
			// skip to next tile.
			// little endian of 0xFF00 & skiptiles
			if nx != width-1 || ny != height-1 {
				outMap.Write([]byte{0x0, 0xFF})
			}
		}
	}
	if skip > 0 {
		outMap.Write([]byte{byte(skip), 0xFF})
		skip = 0
	}

	return nil
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

	if pos.Floor != 7 {
		return fmt.Errorf("TEMPORARILY unsupported initial location. Floor currently must be 7.")
	}

	for floor := 7; floor >= 0; floor-- {
		if err := c.floorDescription(outMap, pos.X+uint16(7-floor-(18/2-1)), pos.Y+uint16(7-floor-(14/2-1)), uint8(floor), 18, 14); err != nil {
			return fmt.Errorf("failed to send floor %d during initialAppearMap: %v", floor, err)
		}
	}

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
