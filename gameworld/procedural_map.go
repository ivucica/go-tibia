package gameworld

import (
	tnet "badc0de.net/pkg/go-tibia/net"
	"bytes"
	"encoding/binary"
	"fmt"

	"github.com/golang/glog"
)

func NewMapDataSource() MapDataSource {
	return &mapDataSource{
		creatures:         map[CreatureID]Creature{},
		generatedMapTiles: map[tnet.Position]MapTile{},
	}
}

//////////////////////////////

type creature struct {
	x, y, z int
	id      CreatureID
}

func (c *creature) GetPos() tnet.Position {
	return tnet.Position{
		X:     uint16(c.x),
		Y:     uint16(c.y),
		Floor: uint8(c.z),
	}
}
func (c *creature) GetID() CreatureID {
	return c.id
}
func (c *creature) GetName() string {
	return "Demo Character"
}


////////////////////////////

type mapDataSource struct {
	creatures map[CreatureID]Creature

	generatedMapTiles map[tnet.Position]MapTile
}
type mapTile struct {
	ground    MapItem
	creatures []Creature

	subscribers []MapTileEventSubscriber
}

type mapItem int

func mapItemOfType(t int) MapItem {
	mi := mapItem(t)
	return &mi
}

///////////////////////////


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
	if idx == 0 && t.ground != nil && t.ground.GetClientType(0) != 0 {
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

// GetClientType returns the protocol specific ID of the item.
//
// If the clientVersion is 0, a non-zero value will be returned anyway,
// but it may not correspond to a sensible client-side version.
func (i *mapItem) GetClientType(clientVersion uint16) int {
	return int(*i)
}
