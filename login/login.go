package login

import (
	"crypto/rsa"
	"encoding/binary"
	"fmt"
	"io"
	"io/ioutil"
	"math/big"
	"net"

	"badc0de.net/pkg/gotserv"

	"github.com/golang/glog"
)

type Login struct {
	pk *rsa.PrivateKey
}

func New(p, q string) (*Login, error) {
	pB, ok := new(big.Int).SetString(p, 10)
	if !ok {
		return nil, fmt.Errorf("login: invalid p", p)
	}
	qB, ok := new(big.Int).SetString(q, 10)
	if !ok {
		return nil, fmt.Errorf("login: invalid q", q)
	}

	p1 := new(big.Int).Sub(pB, big.NewInt(1))
	q1 := new(big.Int).Sub(qB, big.NewInt(1))

	p1q1 := new(big.Int).Mul(p1, q1)
	pubK := rsa.PublicKey{
		E: 65537,
		N: new(big.Int).Mul(pB, qB),
	}
	pk := rsa.PrivateKey{
		Primes:    []*big.Int{pB, qB},
		PublicKey: pubK,
		D:         new(big.Int).ModInverse(big.NewInt(int64(pubK.E)), p1q1),
	}

	pk.Precompute()

	return &Login{
		pk: &pk,
	}, nil
}

func (c *Login) Serve(conn net.Conn) error {
	defer conn.Close()

	msg, err := gotserv.ReadMessage(conn)
	if err != nil {
		return err
	}

	r := io.LimitReader(msg, 1)
	b, err := ioutil.ReadAll(r)
	if err != nil {
		return err
	}
	if len(b) != 1 {
		return fmt.Errorf("no proto id. dropped conn.")
	}
	if b[0] != 0x01 {
		// TODO(ivucica): send error back "wrong protocol"
		// TODO(ivucica): multiplexing on protocol should be done before this
		return fmt.Errorf("wrong protocol: %d", b)
	}

	r = io.LimitReader(msg, 2 /* os */ +2 /* version */ +4*3 /* dat, spr, pic sigs */)

	var connHeader struct {
		OS, Version            uint16
		DatSig, SprSig, PicSig uint32
	}

	err = binary.Read(r, binary.LittleEndian, &connHeader)
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
		Keys    [4]uint32
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
	return nil
}
