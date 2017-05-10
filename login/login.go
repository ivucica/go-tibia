package login

import (
	"bytes"
	"crypto/rsa"
	"encoding/binary"
	"fmt"
	"io"
	"net"

	tnet "badc0de.net/pkg/go-tibia/net"

	"github.com/golang/glog"
)

type LoginServer struct {
	pk *rsa.PrivateKey
}

func NewServer(pk *rsa.PrivateKey) (*LoginServer, error) {
	return &LoginServer{
		pk: pk,
	}, nil
}

func (c *LoginServer) Serve(conn net.Conn, initialMessage *tnet.Message) error {
	defer conn.Close()

	msg := initialMessage

	r := io.LimitReader(msg, 2 /* os */ +2 /* version */ +4*3 /* dat, spr, pic sigs */)

	var connHeader struct {
		OS, Version            uint16
		DatSig, SprSig, PicSig uint32
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
		Version byte
		Keys    [4]uint32
	}
	// TODO(ivucica): restricted reader?
	err = binary.Read(msg, binary.LittleEndian, &keys)
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

	pwd, err := msg.ReadTibiaString()
	if err != nil {
		return fmt.Errorf("pwd read error: %s", err)
	}

	// skip hw spec
	msg.Next(47)

	glog.Infof("acc:%s len(pwd):%d\n", acc, len(pwd))

	////////

	resp := tnet.NewMessage()
	err = MOTD(resp, "1\nHello!")
	if err != nil {
		glog.Errorln("error generating the motd message: ", err)
		return err
	}

	// add checksum and size headers wherever appropriate, and perform
	// XTEA crypto.
	resp, err = resp.Finalize(key)
	if err != nil {
		glog.Errorf("error finalizing login message response: %s", err)
		return err
	}

	// transmit the response
	wr, err := io.Copy(conn, resp)
	if err != nil {
		glog.Errorf("error writing login message response: %s", err)
		return err
	}
	glog.V(2).Infof("written %d bytes", wr)

	///////
	resp = tnet.NewMessage()
	err = CharacterList(resp, []CharacterListEntry{
		CharacterListEntry{
			CharacterName:  "Demo Character",
			CharacterWorld: "Demo World",
			GameFrontend: net.TCPAddr{
				IP:   net.IPv4(127, 0, 0, 1),
				Port: 7171,
			},
		},
	}, 30)
	if err != nil {
		glog.Errorln("error generating the character list message: ", err)
		return err
	}

	// add checksum and size headers wherever appropriate, and perform
	// XTEA crypto.
	resp, err = resp.Finalize(key)
	if err != nil {
		glog.Errorf("error finalizing login message response: %s", err)
		return err
	}

	// transmit the response
	wr, err = io.Copy(conn, resp)
	if err != nil {
		glog.Errorf("error writing login message response: %s", err)
		return err
	}
	glog.V(2).Infof("written %d bytes", wr)

	//////////

	
	return nil
}
