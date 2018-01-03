package gameworld

import (
	tdat "badc0de.net/pkg/go-tibia/dat"
	tnet "badc0de.net/pkg/go-tibia/net"
	"badc0de.net/pkg/go-tibia/otb/items"
	"bytes"
	"crypto/rsa"
	"encoding/binary"
	"fmt"
	"github.com/golang/glog"
	"io"
	"net"
	"time"
)

type GameworldServer struct {
	pk *rsa.PrivateKey
}

// NewServer creates a new GameworldServer which decodes the initial login message using the passed private key.
func NewServer(pk *rsa.PrivateKey) (*GameworldServer, error) {
	return &GameworldServer{
		pk: pk,
	}, nil
}

func (c *GameworldServer) AddItemsOTB(*itemsotb.Items) {
	// TODO(ivucica): Switch to using badc0de.net/pkg/go-tibia/things
	glog.Info("TODO: add items otb")
}
func (c *GameworldServer) AddTibiaDataset(*tdat.Dataset) {
	glog.Info("TODO: add tibia dat")
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

	acc, err := msg.ReadTibiaString()
	if err != nil {
		return fmt.Errorf("account read error: %s", err)
	}

	//pwd, err := msg.ReadTibiaString()
	//if err != nil {
	//	return fmt.Errorf("pwd read error: %s", err)
	//}
	pwd := "?"
	glog.Infof("acc:%s len(pwd):%d\n", acc, len(pwd))

	c.initialAppear(conn, key)

	time.Sleep(5 * time.Second)

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
