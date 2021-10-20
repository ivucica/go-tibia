package gameworld

import (
	"bytes"
	"crypto/rsa"
	"encoding/binary"
	"fmt"
	"io"
	"math/rand"
	"net"
	"time"

	"github.com/golang/glog"
	"github.com/pkg/errors"

	tnet "badc0de.net/pkg/go-tibia/net"
	"badc0de.net/pkg/go-tibia/things"
)

type CreatureID uint32
type Creature interface {
	GetPos() tnet.Position
	SetPos(tnet.Position) error
	GetID() CreatureID
	GetName() string
	GetDir() things.CreatureDirection // TODO: move to tnet? or move tnet.Position to things?
	SetDir(things.CreatureDirection) error
	GetServerType() uint16
	GetOutfitColors() [4]things.OutfitColor
}

var (
	maxCreatureID CreatureID
)

// TODO(ivucica): Move this to map data source
func NewCreatureID() CreatureID {
	maxCreatureID++
	return maxCreatureID
}

type GameworldConnectionID CreatureID
type GameworldConnection struct {
	id GameworldConnectionID

	server       *GameworldServer
	key          [16]byte
	conn         net.Conn
	senderChan   chan *tnet.Message
	receiverChan chan *tnet.Message
	senderQuit   chan struct{}
	mainLoopQuit chan struct{}

	clientVersion uint16
}

func (c *GameworldConnection) PlayerID() (CreatureID, error) {
	// TODO(ivucica): connection ID does not need to be player ID
	return CreatureID(c.id), nil
}

type GameworldServer struct {
	pk     *rsa.PrivateKey
	things *things.Things

	mapDataSource MapDataSource

	LameDuckText string // error to serve during lame duck mode

	// TODO: all these must be per connection
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

	playerID := NewCreatureID()
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
			}
		case <-gwConn.mainLoopQuit:
			break mainLoop
		}
	}
	// TODO: how to safely tell netsender to quit?
	return nil
}

type FightMode uint8

const (
	FightModeUnknown FightMode = iota
	FightModeOffensive
	FightModeBalanced
	FightModeDefensive
)

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

func (c *GameworldConnection) slotEmpty(out *tnet.Message, slot InventorySlot) error {
	out.Write([]byte{0x79, byte(slot)})
	return nil
}
func (c *GameworldConnection) slotItem(out *tnet.Message, slot InventorySlot, item MapItem) error {
	out.Write([]byte{0x78, byte(slot)})
	c.itemDescription(out, item)
	return nil
}

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
