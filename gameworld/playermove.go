package gameworld

import (
	"encoding/binary"
	"fmt"

	tnet "badc0de.net/pkg/go-tibia/net"

	"github.com/golang/glog"
)

func (c *GameworldConnection) playerCancelMove(dir byte) error {
	out := tnet.NewMessage()
	out.Write([]byte{0xB5})
	out.Write([]byte{
		dir, // direction
	})
	c.senderChan <- out
	return nil
}

func (c *GameworldConnection) playerMoveNorth() error {
	outMove := tnet.NewMessage()
	if err := c.playerMoveNorthImpl(outMove); err != nil {
		return err
	}

	c.senderChan <- outMove
	return nil
}

func (c *GameworldConnection) playerMoveEast() error {
	outMove := tnet.NewMessage()
	if err := c.playerMoveEastImpl(outMove); err != nil {
		return err
	}

	c.senderChan <- outMove
	return nil
}

func (c *GameworldConnection) playerMoveSouth() error {
	outMove := tnet.NewMessage()
	if err := c.playerMoveSouthImpl(outMove); err != nil {
		return err
	}

	c.senderChan <- outMove
	return nil
}

func (c *GameworldConnection) playerMoveWest() error {
	outMove := tnet.NewMessage()
	if err := c.playerMoveWestImpl(outMove); err != nil {
		return err
	}

	c.senderChan <- outMove
	return nil
}

func (c *GameworldConnection) playerMoveNorthImpl(outMove *tnet.Message) error {
	pid, err := c.PlayerID()
	if err != nil {
		return err
	}
	player, err := c.server.mapDataSource.GetCreatureByID(pid)
	if err != nil {
		return err
	}
	p := player.GetPos()

	outMove.Write([]byte{0x6D})

	if err := binary.Write(outMove, binary.LittleEndian, p); err != nil {
		return err
	}
	if t, err := c.server.mapDataSource.GetMapTile(p.X, p.Y, p.Floor); err != nil {
		return err
	} else {
		// find source index for creature.
		var itemCount int
		for itemCount = 0; ; itemCount++ {
			// TODO(ivucica): this loop is really silly; expose item count in tile interface
			_, err := t.GetItem(itemCount)
			if err == ItemNotFound {
				break
			}
			if err != nil {
				return err
			}
		}
		for i := 0; ; i++ {
			// TODO(ivucica): allow fetching item stackindex using tile interface
			c, err := t.GetCreature(i)
			if err == CreatureNotFound {
				return fmt.Errorf("creature not found at expected tile (%v / %v)", p, t)
			}
			if err != nil {
				return err
			}
			if c.GetID() == pid {
				outMove.Write([]byte{byte(i + itemCount)})
				glog.Infof("moving from stackpos %d", i+itemCount)
				break
			}
		}

		if err := t.RemoveCreature(player); err != nil {
			return err
		}
	}

	newP := tnet.Position{X: p.X, Y: p.Y - 1, Floor: p.Floor}
	if err := player.SetPos(newP); err != nil {
		return err
	}
	if err := binary.Write(outMove, binary.LittleEndian, newP); err != nil {
		return err
	}

	if t, err := c.server.mapDataSource.GetMapTile(newP.X, newP.Y, newP.Floor); err != nil {
		return err
	} else {
		if err := t.AddCreature(player); err != nil {
			return err
		}
	}

	glog.Infof("oldpos is %v", p)
	glog.Infof("newpos is %v", newP)

	///////////////////////////

	//outMap := tnet.NewMessage()
	outMap := outMove
	outMap.Write([]byte{0x65}) // move north desc

	pos := newP

	startX := pos.X - uint16(c.viewportSizeW()/2-1)
	startY := pos.Y - uint16(c.viewportSizeH()/2-1)

	glog.Infof("playerMoveNorth for player %d at %d %d %d", pid, pos.X, pos.Y, pos.Floor)

	err = c.mapDescription(outMap, startX, startY, int8(pos.Floor), uint16(c.viewportSizeW()), 1)

	return err
}

func (c *GameworldConnection) playerMoveEastImpl(outMove *tnet.Message) error {
	pid, err := c.PlayerID()
	if err != nil {
		return err
	}
	player, err := c.server.mapDataSource.GetCreatureByID(pid)
	if err != nil {
		return err
	}
	p := player.GetPos()

	outMove.Write([]byte{0x6D})

	if err := binary.Write(outMove, binary.LittleEndian, p); err != nil {
		return err
	}
	if t, err := c.server.mapDataSource.GetMapTile(p.X, p.Y, p.Floor); err != nil {
		return err
	} else {
		// find source index for creature.
		var itemCount int
		for itemCount = 0; ; itemCount++ {
			// TODO(ivucica): this loop is really silly; expose item count in tile interface
			_, err := t.GetItem(itemCount)
			if err == ItemNotFound {
				break
			}
			if err != nil {
				return err
			}
		}
		for i := 0; ; i++ {
			// TODO(ivucica): allow fetching item stackindex using tile interface
			c, err := t.GetCreature(i)
			if err == CreatureNotFound {
				return fmt.Errorf("creature not found at expected tile (%v / %v)", p, t)
			}
			if err != nil {
				return err
			}
			if c.GetID() == pid {
				outMove.Write([]byte{byte(i + itemCount)})
				glog.Infof("moving from stackpos %d", i+itemCount)
				break
			}
		}

		if err := t.RemoveCreature(player); err != nil {
			return err
		}
	}

	newP := tnet.Position{X: p.X + 1, Y: p.Y, Floor: p.Floor}
	if err := player.SetPos(newP); err != nil {
		return err
	}
	if err := binary.Write(outMove, binary.LittleEndian, newP); err != nil {
		return err
	}

	if t, err := c.server.mapDataSource.GetMapTile(newP.X, newP.Y, newP.Floor); err != nil {
		return err
	} else {
		if err := t.AddCreature(player); err != nil {
			return err
		}
	}

	glog.Infof("oldpos is %v", p)
	glog.Infof("newpos is %v", newP)

	///////////////////////////

	//outMap := tnet.NewMessage()
	outMap := outMove
	outMap.Write([]byte{0x66}) // move east desc

	pos := newP

	startX := pos.X + uint16(c.viewportSizeW()/2)
	startY := pos.Y - uint16(c.viewportSizeH()/2-1)

	glog.Infof("playerMoveEast for player %d at %d %d %d", pid, pos.X, pos.Y, pos.Floor)

	err = c.mapDescription(outMap, startX, startY, int8(pos.Floor), 1, uint16(c.viewportSizeH()))

	return err
}

func (c *GameworldConnection) playerMoveSouthImpl(outMove *tnet.Message) error {
	pid, err := c.PlayerID()
	if err != nil {
		return err
	}
	player, err := c.server.mapDataSource.GetCreatureByID(pid)
	if err != nil {
		return err
	}
	p := player.GetPos()

	outMove.Write([]byte{0x6D})

	if err := binary.Write(outMove, binary.LittleEndian, p); err != nil {
		return err
	}
	if t, err := c.server.mapDataSource.GetMapTile(p.X, p.Y, p.Floor); err != nil {
		return err
	} else {
		// find source index for creature.
		var itemCount int
		for itemCount = 0; ; itemCount++ {
			// TODO(ivucica): this loop is really silly; expose item count in tile interface
			_, err := t.GetItem(itemCount)
			if err == ItemNotFound {
				break
			}
			if err != nil {
				return err
			}
		}
		for i := 0; ; i++ {
			// TODO(ivucica): allow fetching item stackindex using tile interface
			c, err := t.GetCreature(i)
			if err == CreatureNotFound {
				return fmt.Errorf("creature not found at expected tile (%v / %v)", p, t)
			}
			if err != nil {
				return err
			}
			if c.GetID() == pid {
				outMove.Write([]byte{byte(i + itemCount)})
				glog.Infof("moving from stackpos %d", i+itemCount)
				break
			}
		}

		if err := t.RemoveCreature(player); err != nil {
			return err
		}
	}

	newP := tnet.Position{X: p.X, Y: p.Y + 1, Floor: p.Floor}
	if err := player.SetPos(newP); err != nil {
		return err
	}
	if err := binary.Write(outMove, binary.LittleEndian, newP); err != nil {
		return err
	}

	if t, err := c.server.mapDataSource.GetMapTile(newP.X, newP.Y, newP.Floor); err != nil {
		return err
	} else {
		if err := t.AddCreature(player); err != nil {
			return err
		}
	}

	glog.Infof("oldpos is %v", p)
	glog.Infof("newpos is %v", newP)

	///////////////////////////

	//outMap := tnet.NewMessage()
	outMap := outMove
	outMap.Write([]byte{0x67}) // move south desc

	pos := newP

	startX := pos.X - uint16(c.viewportSizeW()/2-1)
	startY := pos.Y + uint16(c.viewportSizeH()/2)

	glog.Infof("playerMoveSouth for player %d at %d %d %d", pid, pos.X, pos.Y, pos.Floor)

	err = c.mapDescription(outMap, startX, startY, int8(pos.Floor), uint16(c.viewportSizeW()), 1)

	return err
}

func (c *GameworldConnection) playerMoveWestImpl(outMove *tnet.Message) error {
	pid, err := c.PlayerID()
	if err != nil {
		return err
	}
	player, err := c.server.mapDataSource.GetCreatureByID(pid)
	if err != nil {
		return err
	}
	p := player.GetPos()

	outMove.Write([]byte{0x6D})

	if err := binary.Write(outMove, binary.LittleEndian, p); err != nil {
		return err
	}
	if t, err := c.server.mapDataSource.GetMapTile(p.X, p.Y, p.Floor); err != nil {
		return err
	} else {
		// find source index for creature.
		var itemCount int
		for itemCount = 0; ; itemCount++ {
			// TODO(ivucica): this loop is really silly; expose item count in tile interface
			_, err := t.GetItem(itemCount)
			if err == ItemNotFound {
				break
			}
			if err != nil {
				return err
			}
		}
		for i := 0; ; i++ {
			// TODO(ivucica): allow fetching item stackindex using tile interface
			c, err := t.GetCreature(i)
			if err == CreatureNotFound {
				return fmt.Errorf("creature not found at expected tile (%v / %v)", p, t)
			}
			if err != nil {
				return err
			}
			if c.GetID() == pid {
				outMove.Write([]byte{byte(i + itemCount)})
				glog.Infof("moving from stackpos %d", i+itemCount)
				break
			}
		}

		if err := t.RemoveCreature(player); err != nil {
			return err
		}
	}

	newP := tnet.Position{X: p.X - 1, Y: p.Y, Floor: p.Floor}
	if err := player.SetPos(newP); err != nil {
		return err
	}
	if err := binary.Write(outMove, binary.LittleEndian, newP); err != nil {
		return err
	}

	if t, err := c.server.mapDataSource.GetMapTile(newP.X, newP.Y, newP.Floor); err != nil {
		return err
	} else {
		if err := t.AddCreature(player); err != nil {
			return err
		}
	}

	glog.Infof("oldpos is %v", p)
	glog.Infof("newpos is %v", newP)

	///////////////////////////

	//outMap := tnet.NewMessage()
	outMap := outMove
	outMap.Write([]byte{0x68}) // move east desc

	pos := newP

	startX := pos.X - uint16(c.viewportSizeW()/2-1)
	startY := pos.Y - uint16(c.viewportSizeH()/2-1)

	glog.Infof("playerMoveWest for player %d at %d %d %d", pid, pos.X, pos.Y, pos.Floor)

	err = c.mapDescription(outMap, startX, startY, int8(pos.Floor), 1, uint16(c.viewportSizeH()))

	return err
}
