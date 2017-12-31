package gameworld

import (
	tnet "badc0de.net/pkg/go-tibia/net"
	"net"
)

func (c *GameworldServer) initialAppearSelfAppear(outMap *tnet.Message, conn net.Conn, key [16]byte) error {
	outMap.Write([]byte{0x0A}) // self appear
	outMap.Write([]byte{
		0xAA, 0xBB, 0xCC, 0xDD, // own id
		0x00, 0x00, // unkDrawSpeed
		0x00, // canReportBugs
	})
	return nil
}
func (c *GameworldServer) initialAppearMap(outMap *tnet.Message, conn net.Conn, key [16]byte) error {
	outMap.Write([]byte{0x64}) // full map desc
	outMap.Write([]byte{
		0x00, 0x7F, // x
		0x00, 0x7F, // y
		0x07, // floor
	})

	c.floorDescription(outMap, 32768, 32768, 7, 18, 14, 0)

	for skip := 7 * 18 * 14; skip > 0; { // floors 0 through 6 will be empty
		if skip >= 0xFF {
			outMap.Write([]byte{0xFF, 0xFF})
			skip -= 0xFF
		} else {
			outMap.Write([]byte{byte(skip), 0xFF})
			break
		}
	}
	return nil
}

func (c *GameworldServer) floorDescription(outMap *tnet.Message, x, y, z, width, height, offset int) {
	for nx := 0; nx < width; nx++ {
		for ny := 0; ny < height; ny++ {
			outMap.Write([]byte{
				103, 0, // ground
			})

			if nx == width/2 && ny == height/2 {
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
}

func (c *GameworldServer) creatureDescription(outMap *tnet.Message) {
	outMap.Write([]byte{
		0x61, 0x00, // not know n creature thingid
		0x00, 0x00, 0x00, 0x00, // remove
		0xAA, 0xBB, 0xCC, 0xDD,
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
}
