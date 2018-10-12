package gameworld

// This file contains temporary, development-only implementations of some chunks of the game protocol.

import (
	"encoding/binary"

	tnet "badc0de.net/pkg/go-tibia/net"
	"fmt"
)

func (c *GameworldConnection) initialAppearSelfAppear(outMap *tnet.Message) error {
	outMap.Write([]byte{0x0A}) // self appear

	// 0xAA, 0xBB, 0xCC, 0xDD, // own id
	if err := binary.Write(outMap, binary.LittleEndian, c.id); err != nil { // TODO(ivucica): instead of connection ID, send player creature ID (currently the same, not necessarily in the future)
		return err
	}

	outMap.Write([]byte{
		0x00, 0x00, // unkDrawSpeed
		0x00, // canReportBugs
	})
	return nil
}
func (c *GameworldConnection) initialAppearMap(outMap *tnet.Message) error {
	outMap.Write([]byte{0x64}) // full map desc
	outMap.Write([]byte{
		0x00, 0x7F, // x
		0x00, 0x7F, // y
		0x07, // floor
	})

	for floor := 7; floor >= 0; floor-- {
		if err := c.floorDescription(outMap, 32768+(7-floor), 32768+(7-floor), floor, 18, 14); err != nil {
			return fmt.Errorf("failed to send floor %d during initialAppearMap: %v", floor, err)
		}
	}

	return nil
}

func (c *GameworldConnection) creatureDescription(outMap *tnet.Message) error {
	outMap.Write([]byte{
		0x61, 0x00, // not known creature thingid
		0x00, 0x00, 0x00, 0x00, // remove
	})

	// 0xAA, 0xBB, 0xCC, 0xDD, // creature id (this is, currently, player's id)
	if err := binary.Write(outMap, binary.LittleEndian, c.id); err != nil { // TODO(ivucica): instead of connection ID, send player creature ID (currently the same, not necessarily in the future)
		return err
	}

	outMap.Write([]byte{
		0x05, 0x00, 'B', 'o', 'o', '!', '!',
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
