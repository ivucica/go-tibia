package login

import (
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
	/*
		err = binary.Read(msg, binary.LittleEndian, &connHeader.OS)
		err = binary.Read(msg, binary.LittleEndian, &connHeader.Version)
		err = binary.Read(msg, binary.LittleEndian, &connHeader.DatSig)
		err = binary.Read(msg, binary.LittleEndian, &connHeader.SprSig)
		err = binary.Read(msg, binary.LittleEndian, &connHeader.PicSig)*/
	if err != nil {
		return fmt.Errorf("could not read conn header: %s", err)
	}

	glog.Infof("header: %+v", connHeader)

	err = msg.RSADecryptRemainder(c.pk)
	if err != nil {
		return fmt.Errorf("rsa decrypt remainder error: %s", err)
	}

	var keys struct {
		Version byte
		Key     [16]byte
	}
	// TODO(ivucica): restricted reader?
	err = binary.Read(msg, binary.LittleEndian, &keys)
	if err != nil {
		return fmt.Errorf("key read error: %s", err)
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

	resp := tnet.NewMessage()
	MOTD(resp, "Hello!") // TODO(ivucica): error check
	CharacterList(resp, []CharacterListEntry{
		CharacterListEntry{
			CharacterName:  "Demo Character",
			CharacterWorld: "Demo World",
			GameFrontend: net.TCPAddr{
				IP:   net.IPv4(127, 0, 0, 1),
				Port: 7171,
			},
		},
	}, 30) // TODO(ivucica): error check

	// add size
	resp.Finalize(false)

	resp, err = resp.Encrypt(keys.Key)
	if err != nil {
		return err
	}

	// add checksum and size
	resp.Finalize(true)
	io.Copy(conn, resp)

	return nil
}
