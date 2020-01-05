package gameworld

// This file contains temporary, development-only implementations of some chunks of the game protocol.

import (
	"encoding/binary"

	tnet "badc0de.net/pkg/go-tibia/net"
)

func (c *GameworldConnection) initialAppearSelfAppear(outMap *tnet.Message) error {
	outMap.Write([]byte{0x0A}) // self appear

	// 0xAA, 0xBB, 0xCC, 0xDD, // own id
	if err := binary.Write(outMap, binary.LittleEndian, c.id); err != nil { // TODO(ivucica): instead of connection ID, send player creature ID (currently the same, not necessarily in the future)
		return err
	}

	outMap.Write([]byte{
		0x32, 0x00, // unkDrawSpeed
		0x00, // canReportBugs
	})
	return nil
}
