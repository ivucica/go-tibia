package gameworld

import (
	tnet "badc0de.net/pkg/go-tibia/net"
	"badc0de.net/pkg/go-tibia/otb/items"
	"encoding/binary"
	"fmt"
	"io"

	//"github.com/cavaliercoder/go-abs" // int abs is trivial, but *shrug*, this is easy to replace as needed.
	"github.com/golang/glog"
	"github.com/pkg/errors"
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
	GetCount() uint16
}

type MapTileEventSubscriber interface {
}

////////////////////////

func (c *GameworldConnection) viewportSizeW() int8 {
	return 18
}
func (c *GameworldConnection) viewportSizeH() int8 {
	return 14
}
func (c *GameworldConnection) floorGroundLevel() int8 {
	return 7
}
func (c *GameworldConnection) floorBedrockLevel() int8 {
	return 14
}

type singleTileDescription struct {
	pos  tnet.Position
	idx  int
	data *tnet.Message
	err  error
}

func (c *GameworldConnection) mapDescription(outMap *tnet.Message, startX, startY uint16, startFloor int8, width, height uint16) error {
	start := int8(c.floorGroundLevel())
	end := int8(0)
	step := int8(-1)

	if startFloor > c.floorGroundLevel() {
		start = startFloor - 2
		end = c.floorBedrockLevel()
		if int8(startFloor)+2 < end {
			end = int8(startFloor) + 2
		}
		step = 1
	}

	tilesCh := make(chan singleTileDescription)
	descIdx := 0
	for floor := start; floor != end+step; floor += step {
		glog.V(2).Infof("describing floor %d", floor)

		// TODO: handle error from floorDescription
		go func(descIdx int, floor int8) {
			if err := c.floorDescription(tilesCh,
				startX-uint16(floor-startFloor), // TODO(ivucica): fix this calculation
				startY-uint16(floor-startFloor),
				uint8(floor),
				width,
				height, descIdx); err != nil {
				panic(fmt.Errorf("failed to send floor %d: %v %p", floor, err, err))
			}
		}(descIdx, floor)
		descIdx += int(width * height)
	}
	all := make([]singleTileDescription, descIdx) // width*height*uint16(abs.WithTwosComplement(int64(end-start+1))))

	//if len(all) != descIdx {
	//	return fmt.Errorf("len(all) = %d, descIdx = %d, should be same", len(all), descIdx)
	//}

	hadError := false
	for i := 0; i < descIdx; i++ {
		tileDesc := <-tilesCh
		if tileDesc.err != nil {
			glog.Errorf("error in received tile: %v", tileDesc.err)
			// continuing to read the remaining tiles to clear away the channel
		}
		if tileDesc.idx >= descIdx || tileDesc.idx >= len(all) {
			return fmt.Errorf("tried to store at %d in all[%d] - descIdx %d", tileDesc.idx, len(all), descIdx)
		}
		all[tileDesc.idx] = tileDesc
	}
	if hadError {
		return fmt.Errorf("error in at least one received tile")
	}

	var skip int
	for i := 0; i < descIdx; i++ {

		if all[i].data.Len() > 0 {
			// there is data to send.
			//
			// wrap up previous tile by sending tilecount that was skipped.
			//
			// this may be zero, except, of course, if this is the first tile.
			if skip > 0 || i > 0 {
				for skip >= 0 {
					if skip > 255 {
						outMap.Write([]byte{0xFF, 0xFF})
					} else {
						outMap.Write([]byte{byte(skip % 256), 0xFF})
					}
					skip -= 256
					if skip == 0 {
						break
					}
				}
				skip = 0
			}

			io.Copy(outMap, all[i].data)
		} else {
			skip++
		}
	}

	for skip >= 0 {
		if skip > 255 {
			outMap.Write([]byte{0xFF, 0xFF})
		} else {
			outMap.Write([]byte{byte(skip % 256), 0xFF})
		}
		skip -= 256

		if skip == 0 {
			break
		}
	}

	return nil
}

func (c *GameworldConnection) floorDescription(tilesCh chan singleTileDescription, x, y uint16, z uint8, width, height uint16, descIdx int) error {
	for nx := x; nx < x+width; nx++ {
		for ny := y; ny < y+height; ny++ {
			tile, err := c.server.mapDataSource.GetMapTile(nx, ny, z)
			if err != nil {
				return fmt.Errorf("failed to get tile %d %d %d: %v", nx, ny, z, err)
			}

			tileDesc := c.tileDescription(tnet.Position{
				X:     uint16(nx),
				Y:     uint16(ny),
				Floor: uint8(z),
			}, tile, descIdx)
			if tileDesc.err != nil {
				glog.Errorf("failed to send tile desc for %d %d %d: %v", nx, ny, z, tileDesc.err)
				return fmt.Errorf("failed to send tile desc for %d %d %d: %v", nx, ny, z, tileDesc.err)
			}

			tilesCh <- *tileDesc
			descIdx++
		}
	}

	return nil
}

func (c *GameworldConnection) tileDescription(pos tnet.Position, tile MapTile, descIdx int) (tileOut *singleTileDescription) {
	outMap := tnet.NewMessage()
	tileOut = &singleTileDescription{pos: pos, idx: descIdx, data: outMap}

	idx := 0
	for {
		// FIXME: this counts on server order of items matching the client order of items.
		item, err := tile.GetItem(idx)
		if err != nil {
			//_, crErr := tile.GetCreature(0)
			if err == ItemNotFound {
				if idx == 0 {
					glog.V(3).Infof("empty tile %s", tile)
					// TODO: support tiles with no items, but with creatures
					return
				}

				// top of the stack; continue to creatures
				break
			}

			// any other error is an actual error
			tileOut.err = err
			return
		}
		if item == nil {
			glog.Warningf("Bug in map data source: returned item is nil, but error is not ItemNotFound")
			return
		}
		glog.V(3).Infof("sending %s idx %d : %s", tile, idx, item)

		//if idx == 0 {
		//	// little endian of 0xFF00 & skiptiles
		//	outMap.Write([]byte{byte(skip), 0xFF})
		//	skip = 0
		//}

		if err := c.itemDescription(outMap, item); err != nil {
			tileOut.err = err
			return
		}

		idx++
	}

	// add any creatures on this tile
	for idx := 0; ; idx++ {
		if cr, err := tile.GetCreature(idx); err == CreatureNotFound {
			break
		} else if err != nil {
			tileOut.err = err
			return
		} else {
			//glog.Infof("sending %s idx %d : %s", tile, idx, cr)
			if err := c.creatureDescription(outMap, cr); err != nil {
				tileOut.err = err
				return
			}
		}
	}

	return
}

func (c *GameworldConnection) itemDescription(out *tnet.Message, item MapItem) error {
	itemOTBItem := c.server.things.Temp__GetItemFromOTB(item.GetServerType(), c.clientVersion)
	//if itemOTBItem.Group != itemsotb.ITEM_GROUP_GROUND {
	// TODO(ivucica): support tiles with only non-item items or with only creatures (although, does that make sense?)
	//	return emptyTile()
	//}

	itemClientID := c.server.things.Temp__GetClientIDForServerID(item.GetServerType(), c.clientVersion)
	if itemClientID == 0 {
		// some error getting client ID
		return fmt.Errorf("error getting client id for item %d", item.GetServerType())
	}

	out.Write([]byte{
		byte(itemClientID % 256), byte(itemClientID / 256), // item
	})

	if itemOTBItem.Flags&itemsotb.FLAG_STACKABLE != 0 {
		out.Write([]byte{
			byte(item.GetCount()),
		})
	}
	if itemOTBItem.Group == itemsotb.ITEM_GROUP_FLUID || itemOTBItem.Group == itemsotb.ITEM_GROUP_SPLASH || itemOTBItem.Flags&itemsotb.FLAG_CLIENTCHARGES != 0 {
		// either count or fluid color
		out.Write([]byte{
			byte(4),
		})
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
		return errors.Wrap(err, "initialAppearMap: getting player creature")
	}

	pos := creature.GetPos()
	outMap.WriteTibiaPosition(pos)

	glog.V(2).Infof("initialAppearMap for player %d at %d %d %d", playerID, pos.X, pos.Y, pos.Floor)

	startX := pos.X - uint16(c.viewportSizeW()/2-1)
	startY := pos.Y - uint16(c.viewportSizeH()/2-1)

	err = c.mapDescription(outMap, startX, startY, int8(pos.Floor), uint16(c.viewportSizeW()), uint16(c.viewportSizeH()))
	glog.V(2).Infof("initial map sent")

	return err
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
		0x88, 0x00, 0x0a, 0x0a, 0x0a, // 128, 0, 5, 2, 0, // outfit
		0x0a, 0x00, //0, 0, // looktype ex u16
		0, 0, // light level and color
		0x84, 0x03, //100, 0, // step speed
		0, //skull
		0, // party shield
		0, // 0x61, therefore send war emblem
		0x01, //0, // player can walk through
	})

	return nil
}
