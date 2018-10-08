package gameworld

import (
	tnet "badc0de.net/pkg/go-tibia/net"
	"badc0de.net/pkg/go-tibia/things"
	"bytes"
	"crypto/rsa"
	"encoding/binary"
	"fmt"
	"github.com/golang/glog"
	"io"
	"net"
	"time"
)

type Creature interface {
	GetPos() (x, y, z int)
	GetID() uint32
}
type creature struct {
	x, y, z int
	id      uint32
}

func (c *creature) GetPos() (int, int, int) {
	return c.x, c.y, c.z
}
func (c *creature) GetID() uint32 {
	return c.id
}

type GameworldServer struct {
	pk     *rsa.PrivateKey
	things *things.Things

	mapDataSource MapDataSource

}

// NewServer creates a new GameworldServer which decodes the initial login message using the passed private key.
func NewServer(pk *rsa.PrivateKey) (*GameworldServer, error) {
	ds := NewMapDataSource()
	ds.AddCreature(&creature{
		x:  32768 + 18/2,
		y:  32768 + 14/2,
		z:  7,
		id: 0xAA + 0xBB>>8 + 0xCC>>16 + 0xDD>>24,
	})

	return &GameworldServer{
		pk: pk,

		mapDataSource: ds,
	}, nil
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

	c.initialAppear(conn, key)

	conn.SetDeadline(time.Time{}) // Disable deadline

	for {
		msg, err := tnet.ReadMessage(conn)
		if err != nil {
			glog.Errorf("failed to read message: %v", err)
			return err
		}
		msg, err = msg.Decrypt(key)
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
			c.playerCancelMove(conn, key, 0)
			break
		case 0x66: // move east
			c.playerCancelMove(conn, key, 1)
			break
		case 0x67: // move south
			c.playerCancelMove(conn, key, 2)
			break
		case 0x68: // move west
			c.playerCancelMove(conn, key, 3)
			break
		// case 0x69: // stop autowalk
		case 0x6A: // move northeast
			c.playerCancelMove(conn, key, 1)
			break
		case 0x6B: // move southeast
			c.playerCancelMove(conn, key, 2)
			break
		case 0x6C: // move southwest
			c.playerCancelMove(conn, key, 3)
			break
		case 0x6D: // move northwest
			c.playerCancelMove(conn, key, 0)
			break
		}
	}

	return nil
}

func (c *GameworldServer) initialAppear(conn net.Conn, key [16]byte) error {
	outMap := tnet.NewMessage()
	c.initialAppearSelfAppear(outMap, conn, key)
	c.initialAppearMap(outMap, conn, key)
	// add checksum and size headers wherever appropriate, and perform
	// XTEA crypto.
	outMap, err := outMap.Finalize(key)
	if err != nil {
		glog.Errorf("error finalizing map response: %s", err)
		return err
	}

	// transmit the response
	wr, err := io.Copy(conn, outMap)
	if err != nil {
		glog.Errorf("error writing map response: %s", err)
		return err
	}
	glog.V(2).Infof("written %d bytes", wr)
	return nil
}

func (c *GameworldServer) playerCancelMove(conn net.Conn, key [16]byte, dir byte) error {
	out := tnet.NewMessage()
	out.Write([]byte{0xB5})
	out.Write([]byte{
		dir, // direction
	})
	out, err := out.Finalize(key)
	if err != nil {
		glog.Errorf("error finalizing player cancel move: %s", err)
		return err
	}

	// transmit the response
	wr, err := io.Copy(conn, out)
	if err != nil {
		glog.Errorf("error writing map response: %s", err)
		return err
	}
	glog.V(2).Infof("written %d bytes", wr)
	return nil
}
