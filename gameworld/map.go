package gameworld

import (
	tnet "badc0de.net/pkg/go-tibia/net"
	"bytes"
	"encoding/binary"
	"fmt"

	"github.com/golang/glog"
)

var (
	CreatureNotFound error
)

func init() {
	CreatureNotFound = fmt.Errorf("creature not found")
}

type MapDataSource interface {
	GetMapTile(x, y, z int) (MapTile, error)
	GetCreatureByIDBytes(id [4]byte) (Creature, error)
	AddCreature(creature Creature) error
}

type MapTile interface {
	GetItem(idx int) (MapItem, error)
}

type MapItem interface {
	GetType() int
}

func NewMapDataSource() MapDataSource {
	return &mapDataSource{
		creatures: map[uint32]Creature{},
	}
}

type mapDataSource struct {
	creatures map[uint32]Creature
}
type mapTile struct {
	ground mapItem
}
type mapItem int

func (*mapDataSource) GetMapTile(x, y, z int) (MapTile, error) {
	if z == 7 {
		switch ((y + 3) / 2) % 4 {
		case 0:
			return &mapTile{ground: mapItem(103)}, nil
		case 1:
			return &mapTile{ground: mapItem(104)}, nil
		case 2:
			return &mapTile{ground: mapItem(101)}, nil
		case 3:
			return &mapTile{ground: mapItem(100)}, nil
		default:
			return &mapTile{}, nil
		}
	} else {
		if z == 6 && x == 32768+(18/2)-4 && y == 32768+(14/2) {
			glog.Infof("sending 104 at %d %d %d", x, y, z)
			return &mapTile{ground: mapItem(103)}, nil
		}
		return &mapTile{}, nil
	}
}
func (ds *mapDataSource) GetCreatureByIDBytes(idBytes [4]byte) (Creature, error) {
	buf := bytes.NewBuffer(idBytes[:])
	var id uint32
	err := binary.Read(buf, binary.LittleEndian, &id)
	if err != nil {
		return nil, fmt.Errorf("could not decode creature ID from bytes: %v", err)
	}

	if creature, ok := ds.creatures[id]; ok {
		return creature, nil
	}
	return nil, CreatureNotFound
}
func (ds *mapDataSource) AddCreature(c Creature) error {
	ds.creatures[c.GetID()] = c
	//ds.GetMapTile(c.GetPos()).AddCreature(c)
	return nil
}

func (t *mapTile) GetItem(idx int) (MapItem, error) {
	if idx == 0 && int(t.ground) != 0 {
		return &t.ground, nil
	}
	return nil, nil
}

func (i *mapItem) GetType() int {
	return int(*i)
}

////////////////////////

func (c *GameworldServer) floorDescription(outMap *tnet.Message, x, y, z, width, height int) error {
	var skip int
	for nx := x; nx < x+width; nx++ {
		for ny := y; ny < y+height; ny++ {

			tile, err := c.mapDataSource.GetMapTile(nx, ny, z)
			if err != nil {
				return fmt.Errorf("failed to get tile %d %d %d: %v", nx, ny, z, err)
			}
			ground, err := tile.GetItem(0)
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

			// HACK: Spawn player at middle of the maptd
			if nx == x+width/2 && ny == y+height/2 && z == 7 {
				c.creatureDescription(outMap)
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
