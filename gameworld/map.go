package gameworld

import (
	"context"
	"encoding/binary"
	"fmt"
	"io"
	"time"

	"badc0de.net/pkg/go-tibia/gameworld/gwmap"
	tnet "badc0de.net/pkg/go-tibia/net"
	"badc0de.net/pkg/go-tibia/otb/items"

	//"github.com/cavaliercoder/go-abs" // int abs is trivial, but *shrug*, this is easy to replace as needed.
	"github.com/golang/glog"
	"github.com/pkg/errors"
	"golang.org/x/sync/errgroup"
)

var (
	ItemNotFound     error // In case an item is not found, this error is returned.
	CreatureNotFound error // In case a creature is not found, this error is returned.
)

func init() {
	ItemNotFound = fmt.Errorf("item not found")
	CreatureNotFound = fmt.Errorf("creature not found")
}

////// Interfaces //////

// Ideally this block would be empty.
//
// Ideally in Go, we would define types as locally as possible, but this is in a
// separate package to allow sharing with OTBM loader and its tests.
type (
	MapDataSource          = gwmap.MapDataSource
	MapTile                = gwmap.MapTile
	MapTileEventSubscriber = gwmap.MapTileEventSubscriber
	MapItem                = gwmap.MapItem
)

////////////////////////

// viewportSizeW returns the width of the viewport in tiles. It is generally
// fixed for a particular client version (or at least generally static for a
// connection).
func (c *GameworldConnection) viewportSizeW() int8 {
	return 18
}
// viewportSizeH returns the height of the viewport in tiles. It is generally
// fixed for a particular client version (or at least generally static for a
// connection).
func (c *GameworldConnection) viewportSizeH() int8 {
	return 14
}
// floorGroundLevel returns the ground level. It will be fixed for a particular
// client version, and static for a connection. Anything below this (larger
// integers) is considered underground, and rendered accordingly; anything above
// this (smaller integers) is considered above ground, and rendered accordingly.
func (c *GameworldConnection) floorGroundLevel() int8 {
	return 7
}
// floorBedrockLevel returns the bedrock level. It will be fixed for a
// particular client version, and static for a connection. There is nothing
// below this level; this is the lowest level possible (highest level is 0).
func (c *GameworldConnection) floorBedrockLevel() int8 {
	return 14
}

type singleTileDescription struct {
	pos  tnet.Position
	idx  int
	data *tnet.Message
	err  error
}

// mapDescription sends a description of the map to the client. The map is
// described starting from the given position, and extending to the given width
// and height. Multiple floors are sent, and starting and end floor depend on
// whether we are currently underground or above ground. The starting X and Y
// change depending on the floor, because the same X and Y on floors higher
// up is actually rendered with one tile offset to the left and up.
func (c *GameworldConnection) mapDescription(outMap *tnet.Message, startX, startY uint16, startFloor int8, width, height uint16) error {
	glog.Infof("sending %d,%d,%d for %dx%d", startX, startY, startFloor, width, height)
	// Assume we are above ground. Plan to send starting from ground level
	// (typically 7, but may vary), all the way to the top (0).
	start := int8(c.floorGroundLevel())
	end := int8(0)
	step := int8(-1) // Above ground, we are sending from bottom to top.

	if startFloor > c.floorGroundLevel() {
		// Underground. Start from the floor two levels above current floor,
		// and send all the way down to bedrock.
		start = startFloor - 2
		end = c.floorBedrockLevel()
		if int8(startFloor)+2 < end {
			// If we are more than 2 floors above bedrock, send only 2 floors
			// below us.
			end = int8(startFloor) + 2
		}
		// Underground, we are sending from top to bottom.
		step = 1
	}

	ctx, cancel := context.WithTimeout(context.TODO(), 3*time.Second)
	defer cancel()
	group, ctx := errgroup.WithContext(ctx)

	// Calculate total number of tiles to send. Used to create the channel.
	total := int((end + step - start) * step)
	total *= int(width * height)

	// Collect all tiles in a channel, so we can send them in order.
	tilesCh := make(chan singleTileDescription, total)
	descIdx := int(0)
	// Concurrently compute map descriptions for each floor individually.
	for floor := start; floor != end+step; floor += step {
		glog.V(2).Infof("describing floor %d", floor)

		// TODO: reenable concurrently computing the floors.
		//go func(descIdx int, floor int8) {
		func(descIdx int, floor int8) {
			group.Go(func() error {
				if err := c.floorDescription(tilesCh,
					startX-uint16(floor-startFloor), // TODO(ivucica): fix this calculation
					startY-uint16(floor-startFloor),
					uint8(floor),
					width,
					height, descIdx); err != nil {
					return fmt.Errorf("failed to send floor %d: %v %p", floor, err, err)
				}
				return nil
			})
		}(descIdx, floor) // Tell the goroutine this is the floor it is computing, so it can send it back in the channel and the aggregator can put it in the right place.
		descIdx += int(width * height)
	}

	// Wait for all floor descriptions to be computed.
	if err := group.Wait(); err != nil {
		return err
	}

	if descIdx != total {
		panic(fmt.Sprintf("math problem: descIdx %d != total %d", descIdx, total))
	}

	all := make([]singleTileDescription, descIdx) // width*height*uint16(abs.WithTwosComplement(int64(end-start+1))))

	//if len(all) != descIdx {
	//	return fmt.Errorf("len(all) = %d, descIdx = %d, should be same", len(all), descIdx)
	//}

	hadError := false
	// Collect all tiles, and insert them into the 'all' array in the correct
	// order.
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
		// TODO: aggregate 
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

// floorDescription sends a description of a single floor to the client. The
// floor is described starting from the given position, and extending to the
// given width and height.
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

// tileDescription sends a description of a single tile to the client, located
// at the passed position. The tile is described by sending all items on the
// tile, and all creatures on the tile.
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

// itemDescription sends a description of an item to the client. The item is
// described by sending its client ID, and possibly its count or fluid color.
func (c *GameworldConnection) itemDescription(out *tnet.Message, item MapItem) error {
	itemOTBItem := c.server.things.Temp__GetItemFromOTB(item.GetServerType(), c.clientVersion)
	//if itemOTBItem.Group != itemsotb.ITEM_GROUP_GROUND {
	// TODO(ivucica): support tiles with only non-item items or with only creatures (although, does that make sense?)
	//	return emptyTile()
	//}

	itemClientID := c.server.things.Temp__GetClientIDForServerID(item.GetServerType(), c.clientVersion)
	if itemClientID == 0 {
		// some error getting client ID
		return fmt.Errorf("id for item %d is 0", item.GetServerType())
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

func (c *GameworldConnection) TestOnly_InitialAppearMap(outMap *tnet.Message) error {
    return c.initialAppearMap(outMap)
}

// initialAppearMap sends the initial map description to the client. The map is
// described starting from the player's position, and extending to the width and
// height of the viewport.
//
// This function is used to send the initial map description to the client when
// the client first connects to the game server. It is used to send the map
// around the player's position, so the client can render the map around the
// player.
//
// The map is described by sending the player's position. The map is then
// described by sending the map description starting from the player's position,
// and extending to the width and height of the viewport.
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

// creatureDescription sends a description of a creature to the client. The
// creature is described by sending its client ID, name, health, direction,
// outfit, light level, step speed, skull, party shield, war emblem, and
// impassable. The creature is also described by sending its outfit data.
func (c *GameworldConnection) creatureDescription(outMap *tnet.Message, cr Creature) error {
	// TODO(ivucica): support not sending the whole creature (i.e. support something other than 0x61)
	outMap.Write([]byte{
		0x61, 0x00, // not known creature thingid
		0x00, 0x00, 0x00, 0x00, // remove
	})

	// If we are removing and adding the same creature: we are just renaming
	// the creature and changing its type.

	// 0xAA, 0xBB, 0xCC, 0xDD, // creature id (this is, currently, player's id)
	if err := binary.Write(outMap, binary.LittleEndian, cr.GetID()); err != nil {
		return err
	}

	outMap.WriteTibiaString(cr.GetName())

	outMap.Write([]byte{
		100,                // health
		uint8(cr.GetDir()), // dir,
	})

	c.creatureOutfit(outMap, cr)

	outMap.Write([]byte{
		0x00, 0x00, // light level and color
		0x84, 0x03, // step speed, uint16
		0,    //skull
		0,    // party shield
		0,    // 0x61, therefore required to send war emblem in 8.53+
		0x01, // 'impassable', whether players can walk through. 8.53+
	})

	return nil
}

// creatureOutfit sends a description of a creature's outfit to the client. The
// creature's outfit is described by sending its look type, head, body, legs,
// feet, and addons.
func (c *GameworldConnection) creatureOutfit(out *tnet.Message, cr Creature) error {
	itemLook := uint16(0) // look like an item instead? 0 disables
	look := cr.GetServerType()

	if itemLook == 0 && look == 0 {
		return fmt.Errorf("creature %08x's server look type is 0", cr.GetID())
	}

	if itemLook != 0 {
		// TODO(ivucica): does this support more than just look? should full 'itemDescription' be sent?
		out.Write([]byte{0x00, 0x00}) // uint16 zero
		if err := binary.Write(out, binary.LittleEndian, itemLook); err != nil {
			return err
		}
	} else {
		thCr, err := c.server.things.Creature(look, c.clientVersion)
		if err != nil {
			return errors.Wrapf(err, "unsupported creature %08x on scene", cr.GetID())
		}
		cols := cr.GetOutfitColors()
		netOutfit := struct {
			LookType               uint16
			Head, Body, Legs, Feet uint8
			Addons                 uint8
		}{
			LookType: uint16(thCr.ClientID(c.clientVersion)),
			Head:     uint8(cols[0]),
			Body:     uint8(cols[1]),
			Legs:     uint8(cols[2]),
			Feet:     uint8(cols[3]),
			Addons:   uint8(0),
		}
		if netOutfit.LookType == 0 {
			return fmt.Errorf("creature %08x look has clientside id of 0", cr.GetID())
		}
		glog.Infof("sending look type %02x", netOutfit.LookType)
		if err := binary.Write(out, binary.LittleEndian, netOutfit); err != nil {
			return err
		}
	}

	return nil
}
