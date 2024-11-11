package otbm

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"strings"

	"github.com/golang/glog"

	"badc0de.net/pkg/go-tibia/dat"
	"badc0de.net/pkg/go-tibia/gameworld"
	tnet "badc0de.net/pkg/go-tibia/net"
	"badc0de.net/pkg/go-tibia/things"
)

type Map struct {
	gameworld.MapDataSource
	tiles     map[pos]*mapTile
	creatures map[gameworld.CreatureID]gameworld.Creature
	things    *things.Things

	defaultPlayerSpawnPoint pos // temporary variable; ideally this is specified by having player's town ID in config

	desc                       []string
	extSpawnFile, extHouseFile string
}

func (m *Map) String() string {
	return fmt.Sprintf("<map with description: [%s]>", strings.Join(m.desc, "; "))
}

func (m *Map) Private_And_Temp__DefaultPlayerSpawnPoint(c gameworld.CreatureID) tnet.Position {
	pos := m.defaultPlayerSpawnPoint
	return tnet.Position{
		X:     pos.X(),
		Y:     pos.Y(),
		Floor: pos.Floor(),
	}
}

func (m *Map) GetAmbientLight() (dat.DatasetColor, uint8) {
	return gameworld.NightAmbient, gameworld.NightAmbientLevel
}

func (m *Map) AddCreature(c gameworld.Creature) error {
	glog.V(2).Infof("adding creature %d", c.GetID())
	m.creatures[c.GetID()] = c
	if t, err := m.GetMapTile(c.GetPos().X, c.GetPos().Y, c.GetPos().Floor); err != nil {
		return err
	} else {
		glog.V(2).Infof("adding creature to %d %d %d", c.GetPos().X, c.GetPos().Y, c.GetPos().Floor)

		// HACK: tile has no ground? add it.
		// REMOVE THIS once maps are correctly loaded.
		if i, err := t.GetItem(0); err != nil || i == nil {
			glog.V(2).Info("  but first adding some ground for the creature")
			item := mapItem{
				ancestorMap:   m,
				parentTile:    t.(*mapTile),
				parentItem:    nil,
				otbItemTypeID: 100,
			}
			t.(*mapTile).ground = item
		}

		return t.AddCreature(c)
	}
}

func (m *Map) GetMapTile(x, y uint16, z uint8) (gameworld.MapTile, error) {
	pos := posFromCoord(x, y, z)
	if t, ok := m.tiles[pos]; ok { //tnet.Position{x, y, z}]; ok {
		return t, nil
	}
	//return fmt.Errorf("tile not found") // TODO(ivucica): we should not return a tile
	return &mapTile{parent: m, ownPos: pos}, nil
}

func (m *Map) GetCreatureByIDBytes(idBytes [4]byte) (gameworld.Creature, error) {
	buf := bytes.NewBuffer(idBytes[:])
	var id gameworld.CreatureID
	err := binary.Read(buf, binary.LittleEndian, &id)
	if err != nil {
		return nil, fmt.Errorf("could not decode creature ID from bytes: %v", err)
	}

	return m.GetCreatureByID(id)
}
func (m *Map) GetCreatureByID(id gameworld.CreatureID) (gameworld.Creature, error) {
	if creature, ok := m.creatures[id]; ok {
		return creature, nil
	}
	return nil, gameworld.CreatureNotFound
}

func (m *Map) RemoveCreatureByID(id gameworld.CreatureID) error {
	c, err := m.GetCreatureByID(id)
	if err != nil {
		if err == gameworld.CreatureNotFound {
			return nil
		}
	}

	delete(m.creatures, id)

	if t, err := m.GetMapTile(c.GetPos().X, c.GetPos().Y, c.GetPos().Floor); err != nil {
		return err
	} else {
		glog.V(2).Infof("deleting creature from %d %d %d", c.GetPos().X, c.GetPos().Y, c.GetPos().Floor)
		return t.RemoveCreature(c)
	}
}
