package gameworld

import (
	"bytes"
	"crypto/rsa"
	"encoding/binary"
	"fmt"
	"io"
	"math/rand"
	"net"
	"strings"
	"time"

	"github.com/golang/glog"
	"github.com/pkg/errors"

	"badc0de.net/pkg/go-tibia/gameworld/gwmap"
	tnet "badc0de.net/pkg/go-tibia/net"
	"badc0de.net/pkg/go-tibia/paths"
	"badc0de.net/pkg/go-tibia/things"
	"badc0de.net/pkg/go-tibia/xmls"
)

type (
	CreatureID = gwmap.CreatureID
	Creature   = gwmap.Creature
)

// Creature types are bits that determine what is the type of a particular
// creature. These flags need to be consistent between all data sources.
// Currently they are flags, but they might need to be ranges in the future.
//
// They are used in creature IDs and *affect client behavior*.
//
// Creature IDs within a range might be used for routing RPCs in the future.
//
// BUG(ivucica): The creature type "NPC" has a different value between OT and
// TFS.
//
// BUG(ivucica): The Forgotten Client has ranges < 0x400000000 for players,
// < 0x80000000 for monsters and everything else for NPCs. It also says that
// for >= 9.10 a creature will get a server-assigned type: 0 for players, 1
// for monsters, 2 for NPCs and 3 for summoned creatures (3 is presumed and
// needs to be checked in servers); and on >= 11.21 a summoned creature will
// have the owner's ID determined and will client-side get the assigned type
// of 4.
// https://github.com/opentibiabr/The-Forgotten-Client/blob/8b7979619ea76bc29581122440d09f241afc175d/src/protocolgame.cpp#L8492-L8509
type CreatureType uint32

// Note: wrong constants may affect client behavior (right click options etc).
// The behavior with various values should be validated against various clients.
const (
	// The ID of this creature determines it to be a player.
	//
	// Same in OT and TFS: https://github.com/otland/forgottenserver/blob/973855c3e0a60461117b55f248ad14bab6630780/src/player.cpp#L40
	CreatureTypePlayer = CreatureType(0x10000000)

	// The ID of this creature determines it to be an NPC.
	//
	// Not in OT, value 0x20000000 in TFS: https://github.com/otland/forgottenserver/blob/973855c3e0a60461117b55f248ad14bab6630780/src/npc.cpp#L15
	//
	// TFS v1.2 from 2016 used 0x80000000: https://github.com/otland/forgottenserver/blob/afeea42cee45a176aeccb330733a606e3bdcb64a/src/npc.cpp#L29C27-L29C37
	CreatureTypeNPC = CreatureType(0x20000000)

	// The ID of this creature determines it to be a monster.
	//
	// Not using TFS value 0x21000000 in monster.cpp, choosing value in OT
	// and in TFS v1.2: https://github.com/opentibia/server/blob/33a81ef95a9b407533e0a7ce48aff12204b4f3b1/src/actor.h#L68
	CreatureTypeMonster = CreatureType(0x40000000)
)

// String implements the stringer type, returning all types that this creature
// satisfies. Usually this should be just one type.
func (ct CreatureType) String() string {
	var types []string
	if ct&CreatureTypePlayer != 0 {
		types = append(types, "player")
	}
	if ct&CreatureTypeNPC != 0 {
		types = append(types, "npc")
	}
	if ct&CreatureTypeMonster != 0 {
		types = append(types, "monster")
	}
	if len(types) == 0 {
		return "(generic creature)"
	}
	return strings.Join(types, ", ")
}

var (
	maxCreatureIDPlayer      CreatureID
	maxCreatureIDNPC         CreatureID
	maxCreatureIDMonster     CreatureID
	maxCreatureIDDefaultPool CreatureID
)

// NewCreatureID creates a new creature ID, unique across all data sources,
// and determinable to be a player, NPC, or monster.
//
// BUG(ivucica): Move this to map data source
func NewCreatureID(kind CreatureType) CreatureID {
	switch kind {
	case CreatureTypePlayer:
		maxCreatureIDPlayer++
		return maxCreatureIDPlayer | CreatureID(kind)
	case CreatureTypeNPC:
		maxCreatureIDNPC++
		return maxCreatureIDNPC | CreatureID(kind)
	case CreatureTypeMonster:
		maxCreatureIDMonster++
		return maxCreatureIDMonster | CreatureID(kind)
	default:
		glog.Warningf("creature added with unknown type; client behavior unpredictable (may affect right-click options, e.g.)")
		maxCreatureIDDefaultPool++
		return maxCreatureIDDefaultPool
	}
}

// GameworldConnectionID is the ID of this connection, and at this time will
// match the player ID. This can change in the future.
type GameworldConnectionID CreatureID

// GameworldConnection encapsulates a single active connection (not necessarily
// a session, if we introduce such a concept).
type GameworldConnection struct {
	id GameworldConnectionID

	server       *GameworldServer   // Parent server owning this connection.
	key          [16]byte           // XTEA key.
	conn         net.Conn           // TCP connection (which could be an io.ReadWriteCloser).
	senderChan   chan *tnet.Message // Put a message into this channel to have it sent to the client on this connection.
	receiverChan chan *tnet.Message // Any messages received from the client on this connection will be put into this channel.
	senderQuit   chan struct{}      // Signal to quit the sender goroutine.
	mainLoopQuit chan struct{}      // Signal to quit the main loop goroutine.

	clientVersion uint16
}

// PlayerID returns the player ID for this connection.
//
// Currently it is the same as the creature ID of the player.
func (c *GameworldConnection) PlayerID() (CreatureID, error) {
	// TODO(ivucica): connection ID does not need to be player ID
	return CreatureID(c.id), nil
}

// GameworldServer encapsulates a single gameworld server with all of the
// active connections. This particular implementation does not enable scaling
// the frontends, since all the connections are stored in a single local
// non-distributed map. This implementation also allows only a single map
// data source.
//
// The actual gameworld protocol implementation is currently in this type, not
// in individual connections; the connections just store the metadata for a
// particular network connection from a player.
type GameworldServer struct {
	pk     *rsa.PrivateKey // private key for RSA encryption (has to match the public key in the client)
	things *things.Things  // things registry

	mapDataSource MapDataSource // data source for map information; can be a multiplexer (combining remote RPCs, local map etc) or just a local map

	LameDuckText string // error to serve during lame duck mode

	// TODO: all these must be per network connection
	connections map[GameworldConnectionID]*GameworldConnection
}

// NewServer creates a new GameworldServer which decodes the initial login message using the passed private key.
func NewServer(pk *rsa.PrivateKey) (*GameworldServer, error) {
	return &GameworldServer{
		pk: pk,

		connections: make(map[GameworldConnectionID]*GameworldConnection),
	}, nil
}

// SetMapDataSource sets the data source for map information such as tiles, items
// on tiles, creatures present, etc.
func (c *GameworldServer) SetMapDataSource(ds MapDataSource) error {
	c.mapDataSource = ds
	return nil
}

// SetThings sets the thing registry to the passed value. It's used to refer to a
// combination of items.otb, Tibia.dat and Tibia.spr from the gameworld.
//
// It's not constructed by GameworldServer as the same registry may be used for
// other servers (such as a web server).
func (c *GameworldServer) SetThings(t *things.Things) error {
	c.things = t
	return nil
}

func (c *GameworldConnection) TestOnly_Setter(clientVersion uint16, gws *GameworldServer, id GameworldConnectionID) {
	c.clientVersion = clientVersion
	c.server = gws
	c.id = id
}

// Serve begins serving the gameworld protocol on the accepted network connection.
//
// User of this method needs to bring their own listening schema and accept the connection,
// then pass on the control to this method.
//
// User also needs to transmit the initial gameworld message which the server sends.
func (c *GameworldServer) Serve(conn net.Conn, initialMessage *tnet.Message) error {
	defer conn.Close()

	msg := initialMessage

	r := io.LimitReader(msg, 2 /* os */ +2 /* version */ +4*3 /* dat, spr, pic sigs */)

	var connHeader struct {
		OS, Version uint16
		//		DatSig, SprSig, PicSig uint32
	}

	err := binary.Read(r, binary.LittleEndian, &connHeader)
	if err != nil {
		return fmt.Errorf("could not read conn header: %s", err)
	}

	glog.V(2).Infof("header: %+v", connHeader)
	err = msg.RSADecryptRemainder(c.pk)
	if err != nil {
		return fmt.Errorf("rsa decrypt remainder error: %s", err)
	}

	var keys struct {
		Version byte // is this name right?
		Keys    [4]uint32
	}
	r = io.LimitReader(msg, 1+4*4)
	err = binary.Read(r, binary.LittleEndian, &keys)
	if err != nil {
		return fmt.Errorf("key read error: %s", err)
	}

	// XTEA in Go is bigendian-only. It treats the key as a single
	// 128-bit integer, stored as bigendian. It then explodes it
	// into [4]uint32.
	//
	// We need to flip the order of bytes in the key, otherwise
	// we would quite easily be able to use Keys [16]byte and be
	// done with it.
	key := [16]byte{}
	keyB := &bytes.Buffer{}
	err = binary.Write(keyB, binary.BigEndian, keys.Keys)
	if err != nil {
		return fmt.Errorf("could not convert binary order of keys: %s", err)
	}
	for i := range key {
		key[i] = keyB.Bytes()[i]
	}

	isGM := byte(0)
	r = io.LimitReader(msg, 1)
	err = binary.Read(r, binary.LittleEndian, &isGM)
	if err != nil {
		return fmt.Errorf("could not read isGM: %v", err)
	}

	acc, err := msg.ReadTibiaString()
	if err != nil {
		return fmt.Errorf("account read error: %s", err)
	}

	char, err := msg.ReadTibiaString()
	if err != nil {
		return fmt.Errorf("character read error: %s", err)
	}

	pwd, err := msg.ReadTibiaString()
	if err != nil {
		return fmt.Errorf("pwd read error: %s", err)
	}
	//pwd := "?"
	glog.Infof("acc:%s char:%s len(pwd):%d isGM:%d\n", acc, char, len(pwd), isGM)

	playerID := NewCreatureID(CreatureTypePlayer)
	gwConn := &GameworldConnection{}
	gwConn.clientVersion = connHeader.Version
	gwConn.server = c
	gwConn.conn = conn
	gwConn.key = key
	gwConn.id = GameworldConnectionID(playerID)

	if c.LameDuckText != "" {
		out := tnet.NewMessage()

		glog.Infof("rejection: sending %s", c.LameDuckText)

		out.Write([]byte{0x14}) // there's also 0x0A
		out.WriteTibiaString(c.LameDuckText)

		msg, err := out.Finalize(gwConn.key)
		if err != nil {
			glog.Errorf("error finalizing message: %s", err)
			// TODO: c.quitChan <- struct{} so we close the connection
			return err
		}

		// transmit the response
		wr, err := io.Copy(gwConn.conn, msg)
		if err != nil {
			glog.Errorf("error writing message: %s", err)
			// TODO: c.quitChan <- struct{} so we close the connection
			return err
		}
		glog.Infof("rejection: written %d bytes", wr)

		gwConn.conn.Close() // actually close more nicely

		return nil
	}

	return c.serveGame(conn, initialMessage, gwConn, playerID)
}

func (c *GameworldServer) serveGame(conn net.Conn, initialMessage *tnet.Message, gwConn *GameworldConnection, playerID CreatureID) error {

	defPos := c.mapDataSource.Private_And_Temp__DefaultPlayerSpawnPoint(playerID)

	playerCreature := &creature{
		pos: defPos,
		id:  playerID, //0xAA + 0xBB>>8 + 0xCC>>16 + 0xDD>>24,

		dir:  things.CreatureDirectionSouth,
		look: 129,
		col: [4]things.OutfitColor{
			things.OutfitColor(rand.Int() % things.OutfitColorCount()),
			things.OutfitColor(rand.Int() % things.OutfitColorCount()),
			things.OutfitColor(rand.Int() % things.OutfitColorCount()),
			things.OutfitColor(rand.Int() % things.OutfitColorCount()),
		},
	}
	c.mapDataSource.AddCreature(playerCreature)

	cols := playerCreature.GetOutfitColors()
	glog.Infof("  -> colors %d %d %d %d", cols[0], cols[1], cols[2], cols[3])

	gwConn.senderQuit = make(chan struct{})
	gwConn.senderChan = make(chan *tnet.Message)
	// TODO: how to clean up and close channels safely?
	//defer func() { close(c.senderChan) ; close(c.senderQuit) }()
	go gwConn.networkSender()

	if err := gwConn.initialAppear(); err != nil {
		return fmt.Errorf("failed to send initial appear: %v", err)
	}
	conn.SetDeadline(time.Time{}) // Disable deadline

	gwConn.mainLoopQuit = make(chan struct{})
	gwConn.receiverChan = make(chan *tnet.Message)
	go gwConn.networkReceiver()

	defer c.mapDataSource.RemoveCreatureByID(playerID)

	c.connections[gwConn.id] = gwConn

mainLoop:
	for {
		glog.Infof("pending event on e.g. receiver chan")
		select {
		case encryptedMsg := <-gwConn.receiverChan:
			glog.Infof("received message on receiver chan")
			msg, err := encryptedMsg.Decrypt(gwConn.key)
			if err != nil {
				glog.Errorf("failed to decrypt message: %v", err)
				return err
			}
			glog.Infof("decrypted message: %d", msg.Len())

			msgType, err := msg.ReadByte()
			if err != nil {
				glog.Errorf("error reading msg type: %v", err)
				return err
			}

			glog.Infof("received message: %x", msgType)
			switch msgType {
			case 0x14: // logout
				return nil
			case 0x65: // move north
				if err := gwConn.playerMoveNorth(); err != nil {
					glog.Errorf("error moving player to the north: %v", err)
					continue mainLoop
				}
				break
			case 0x66: // move east
				if err := gwConn.playerMoveEast(); err != nil {
					glog.Errorf("error moving player to the east: %v", err)
					continue mainLoop
				}
				break
			case 0x67: // move south
				if err := gwConn.playerMoveSouth(); err != nil {
					glog.Errorf("error moving player to the south: %v", err)
					continue mainLoop
				}
				break
			case 0x68: // move west
				if err := gwConn.playerMoveWest(); err != nil {
					glog.Errorf("error moving player to the west: %v", err)
					continue mainLoop
				}
				break
			// case 0x69: // stop autowalk
			case 0x6A: // move northeast
				gwConn.playerCancelMove(1)
				break
			case 0x6B: // move southeast
				gwConn.playerCancelMove(2)
				break
			case 0x6C: // move southwest
				gwConn.playerCancelMove(3)
				break
			case 0x6D: // move northwest
				gwConn.playerCancelMove(0)
				break
			case 0x96: // say
				chatType, err := msg.ReadByte()
				if err != nil {
					glog.Errorf("error reading chat type: %v", err)
					continue mainLoop
				}
				switch chatType {
				case 0x01: // say
					chatText, err := msg.ReadTibiaString()
					if err != nil {
						glog.Errorf("error reading chat text: %v", err)
						continue mainLoop
					}
					glog.Infof("%v: %v", "Demo Character", chatText)
					playerCr, err := c.mapDataSource.GetCreatureByID(playerID)
					if err != nil {
						glog.Errorf("error getting player creature by id: %v", err)
						continue mainLoop
					}

					for _, otherGwConn := range c.connections {
						out := tnet.NewMessage()
						out.Write([]byte{0xAA})
						out.Write([]byte{0x00, 0x00, 0x00, 0x00}) // unkSpeak
						out.WriteTibiaString("Demo Character")
						out.Write([]byte{0x01, 0x00}) // level
						out.Write([]byte{0x01})       // type - i.e. 'say' in this case
						out.WriteTibiaPosition(playerCr.GetPos())
						out.WriteTibiaString(chatText)
						//gwConn.senderChan <- out
						go func(otherGwConn *GameworldConnection, msg *tnet.Message) {
							otherGwConn.senderChan <- out
						}(otherGwConn, out)
					}
				}
			case 0xA0: // set fight modes
				var fightMode FightMode
				var chaseMode ChaseMode
				var safeMode uint8 // ?
				if fightModeB, err := msg.ReadByte(); err != nil {
					return err
				} else {
					fightMode = FightMode(fightModeB)
				}

				if chaseModeB, err := msg.ReadByte(); err != nil {
					return err
				} else {
					chaseMode = ChaseMode(chaseModeB)
				}

				if safeModeB, err := msg.ReadByte(); err != nil {
					return err
				} else {
					safeMode = safeModeB
				}

				glog.Infof("fight mode: %v; chase mode: %v; safe mode: %02x", fightMode, chaseMode, safeMode)
			case 0xD2: // request outfit window
				out := tnet.NewMessage()
				if err := gwConn.outfitWindow(out); err != nil {
					glog.Errorf("could not provide outfit window: %v", err)
				} else {
					gwConn.senderChan <- out
				}
			}
		case <-gwConn.mainLoopQuit:
			break mainLoop
		}
	}
	// TODO: how to safely tell netsender to quit?
	return nil
}

// FightMode encapsulates an individual player's intended requested fight stance.
type FightMode uint8

const (
	FightModeUnknown FightMode = iota
	FightModeOffensive
	FightModeBalanced
	FightModeDefensive
)

// String implements the stringer method. It's just encoding the enum type.
func (m FightMode) String() string {
	switch m {
	case FightModeOffensive:
		return "FightModeOffensive"
	case FightModeBalanced:
		return "FightModeBalanced"
	case FightModeDefensive:
		return "FightModeDefensive"
	default:
		return fmt.Sprintf("invalid fight mode %02x", uint8(m))
	}
}

// ChaseMode encapsulates the player's requested behavior when it comes to
// chasing the targeted creature.
type ChaseMode uint8

const (
	ChaseModeStand ChaseMode = iota
	ChaseModeChase
)

func (m ChaseMode) String() string {
	switch m {
	case ChaseModeStand:
		return "ChaseModeStand"
	case ChaseModeChase:
		return "ChaseModeChase"
	default:
		return fmt.Sprintf("invalid chase mode %02x", uint8(m))
	}
}

// networkReceiver receives messages from the client. It is a goroutine that
// reads messages from the network connection and puts them into the receiverChan
// for the main loop to process.
func (c *GameworldConnection) networkReceiver() error {
	// TODO: how to safely tell main loop to quit?
	for {
		msg, err := tnet.ReadMessage(c.conn)
		if err != nil {
			glog.Errorf("failed to read message: %v", err)
			// TODO: c.quitChan <- struct{} so we close the connection
			return err
		}
		glog.Infof("dispatching message to receiver chan")
		c.receiverChan <- msg
		glog.Infof("dispatched message to receiver chan")
	}
}

// networkSender sends messages to the client. It is a goroutine that reads
// messages from the senderChan and sends them to the client over the network,
// uninterrupted so that the messages don't arrive interrupted by other messages.
//
// This is also the point at which the messages are finalized, i.e. they are
// encrypted, and have their size header added.
func (c *GameworldConnection) networkSender() error {
	// TODO: how to safely tell main loop to quit?
	for {
		select {
		case rawMsg := <-c.senderChan:
			glog.Infof("sending a message")
			// add checksum and size headers wherever appropriate, and perform
			// XTEA crypto.
			msg, err := rawMsg.Finalize(c.key)
			if err != nil {
				glog.Errorf("error finalizing message: %s", err)
				// TODO: c.quitChan <- struct{} so we close the connection
				return err
			}

			// transmit the response
			wr, err := io.Copy(c.conn, msg)
			if err != nil {
				glog.Errorf("error writing message: %s", err)
				// TODO: c.quitChan <- struct{} so we close the connection
				return err
			}
			glog.V(2).Infof("written %d bytes", wr)
		case <-c.senderQuit:
			return nil
		}
	}
}

// initialAppear sends the initial appear message to the client. This message
// is sent when the client first connects to the server and is used to present
// the player's character, inventory, skills, etc. to the client, as well as
// the map.
func (c *GameworldConnection) initialAppear() error {
	outMap := tnet.NewMessage()
	c.initialAppearSelfAppear(outMap)
	err := c.initialAppearMap(outMap)
	if err != nil {
		return fmt.Errorf("initialAppear(): %v", err)
	}

	for slot := InventorySlotFirst; slot <= InventorySlotLast; slot++ {
		if slot == InventorySlotHead {
			c.slotItem(outMap, slot, mapItemOfType(104))
			continue
		}
		if err := c.slotEmpty(outMap, slot); err != nil {
			return fmt.Errorf("initialAppear(): slot %v: %s", slot, err.Error())
		}
	}

	if err := c.playerStats(outMap); err != nil {
		return err
	}
	if err := c.playerSkills(outMap); err != nil {
		return err
	}
	if err := c.worldLight(outMap); err != nil {
		return err
	}

	var playerID CreatureID
	if playerIDI, err := c.PlayerID(); err != nil {
		return err
	} else {
		playerID = playerIDI
	}

	if err := c.creatureLight(outMap, playerID); err != nil {
		return err
	}

	// TODO: send vip list

	if err := c.playerIcons(outMap); err != nil {
		return err
	}

	c.senderChan <- outMap
	return nil
}

// InventorySlot is an enum type describing one of the slots where a player can
// directly insert an item onto the character.
type InventorySlot byte

const (
	InventorySlotUnknown  InventorySlot = iota // 0
	InventorySlotHead                          // 1
	InventorySlotNecklace                      // 2
	InventorySlotBackpack                      // 3
	InventorySlotArmor                         // 4
	InventorySlotRight                         // 5
	InventorySlotLeft                          // 6
	InventorySlotLegs                          // 7
	InventorySlotFeet                          // 8
	InventorySlotRing                          // 9
	InventorySlotAmmo                          // A

	InventorySlotFirst = InventorySlotHead
	InventorySlotLast  = InventorySlotAmmo
)

// slotEmpty sends a message to the client to clear an inventory slot.
func (c *GameworldConnection) slotEmpty(out *tnet.Message, slot InventorySlot) error {
	out.Write([]byte{0x79, byte(slot)})
	return nil
}

// slotItem sends a message to the client to present an item in a particular
// inventory slot.
func (c *GameworldConnection) slotItem(out *tnet.Message, slot InventorySlot, item MapItem) error {
	out.Write([]byte{0x78, byte(slot)})
	c.itemDescription(out, item)
	return nil
}

// playerStats sends the player's statistics to the client. This includes the
// player's health, mana, experience, level, magic level, soul points, stamina,
// and capacity.
func (c *GameworldConnection) playerStats(out *tnet.Message) error {
	out.Write([]byte{0xA0})
	stats := struct {
		Health, MaxHealth uint16
		Capacity          uint32 // capacity * 100
		Experience        int32  // if negative, send zero
		Level             uint16
		LevelPercent      uint8
		Mana, MaxMana     uint16
		MagicLevel        uint8
		MagicLevelPercent uint8
		Soul              uint8
		StaminaMinutes    uint16
	}{
		Health:            100,
		MaxHealth:         100,
		Capacity:          500 * 100,
		Level:             1,
		LevelPercent:      5,
		Mana:              50,
		MaxMana:           100,
		MagicLevel:        2,
		MagicLevelPercent: 15,
		Soul:              48,
		StaminaMinutes:    500,
	}

	if stats.Experience < 0 {
		stats.Experience = 0
	}

	if err := binary.Write(out, binary.LittleEndian, stats); err != nil {
		return err
	}
	return nil
}

// Skill describes an individual trainable skill that the character can level
// over time.
type Skill byte

const (
	SkillFist Skill = iota // 0
	SkillClub
	SkillSword
	SkillAxe
	SkillDistance
	SkillShield
	SkillFishing

	SkillFirst = SkillFist
	SkillLast  = SkillFishing
)

func (c *GameworldConnection) playerSkills(out *tnet.Message) error {
	out.Write([]byte{0xA1})
	skillSet := struct {
		FistLevel, FistPercent         uint8
		ClubLevel, ClubPercent         uint8
		SwordLevel, SwordPercent       uint8
		AxeLevel, AxePercent           uint8
		DistanceLevel, DistancePercent uint8
		ShieldLevel, ShieldPercent     uint8
		FishingLevel, FishingPercent   uint8
	}{
		FistLevel:     1,
		FistPercent:   95,
		ClubLevel:     1,
		SwordLevel:    1,
		AxeLevel:      1,
		DistanceLevel: 1,
		ShieldLevel:   1,
		FishingLevel:  1,
	}

	if err := binary.Write(out, binary.LittleEndian, skillSet); err != nil {
		return err
	}
	return nil
}

func (c *GameworldConnection) worldLight(out *tnet.Message) error {
	out.Write([]byte{0x82})
	light := struct {
		Level uint8
		Color uint8
	}{
		Level: 40, // LIGHT_LEVEL_NIGHT
		Color: 0xD7,
	}
	if err := binary.Write(out, binary.LittleEndian, light); err != nil {
		return err
	}
	return nil
}

func (c *GameworldConnection) creatureLight(out *tnet.Message, creature CreatureID) error {
	out.Write([]byte{0x8D})
	light := struct {
		Creature uint32
		Level    uint8
		Color    uint8
	}{
		Creature: uint32(creature),
		Level:    2,
		Color:    45,
	}
	if err := binary.Write(out, binary.LittleEndian, light); err != nil {
		return err
	}
	return nil
}

func (c *GameworldConnection) playerIcons(out *tnet.Message) error {
	// TODO: send actual flags for various icons
	out.Write([]byte{0xA2, 0x00, 0x00})
	return nil
}

// outfitWindow sends the outfit window to the client. The outfit window is a
// window that allows the player to select a new outfit for their character.
// The outfit window is opened by the client when the player right-clicks on
// their character and selects "Outfit". The outfit window is populated with
// outfits that the player can select from. The outfits are read from the
// outfits.xml file.
func (c *GameworldConnection) outfitWindow(out *tnet.Message) error {
	playerID, err := c.PlayerID()
	if err != nil {
		return errors.Wrap(err, "outfitWindow: attempted to open a window while playerID failed")
	}
	playerCreature, err := c.server.mapDataSource.GetCreatureByID(playerID)
	if err != nil {
		return errors.Wrap(err, "outfitWindow: getting player creature")
	}

	// TODO(ivucica): move reading the file to process startup; cache in GameworldServer or so
	f, err := paths.Open("outfits.xml")
	if err != nil {
		return errors.Wrap(err, "outfitWindow: failed to open outfits.xml")
	}
	defer f.Close()

	outfits, err := xmls.ReadOutfits(f)
	if err != nil {
		return errors.Wrap(err, "outfitWindow: failed to parse outfits.xml")
	}

	// TODO(ivucica): actually query player's characteristics to determine which looks are permitted
	permittedTypes := []string{"male", "femalecm"}
	hasPremium := true
	unlockedIDs := []int{
		12, // pirate
	}

	var looks []xmls.OutfitListEntry

	for _, outfit := range outfits.Outfit {
		if outfit.Default == "" {
			outfit.Default = "1"
		}

		// If not unlocked by default, player must have unlocked it somehow.
		if outfit.Default == "0" {
			unlocked := false
			for _, i := range unlockedIDs {
				if i == outfit.ID {
					unlocked = true
				}
			}
			if !unlocked {
				continue
			}
		}

		// Must have premium for premium outfits.
		if !(outfit.Premium == 0 || hasPremium) {
			continue
		}

		// Otherwise we can proceed.
		for _, look := range outfit.List {
			permitted := false
			for _, typ := range permittedTypes {
				if typ == string(look.Type) {
					permitted = true
					break
				}
			}
			if permitted {
				looks = append(looks, look)
			}
		}
	}

	if len(looks) > 25 { // max number of outfits allowed is 25
		looks = looks[:25]
	}

	out.WriteByte(0xC8)

	if err := c.creatureOutfit(out, playerCreature); err != nil {
		return err
	}

	if len(looks) == 0 {
		// TODO(ivucica): send something default
		return fmt.Errorf("no outfits permitted for current player")
	}

	out.WriteByte(byte(len(looks))) // number of wearable outfits
	for _, look := range looks {
		if err := binary.Write(out, binary.LittleEndian, uint16(look.LookType)); err != nil {
			return err
		}
		out.WriteTibiaString(look.Name)
		out.Write([]byte{0}) // addon count
		// TODO(ivucica): support addon count (by examining dat file)
	}

	return nil
}
