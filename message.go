package gotserv

import (
	"bytes"
	"crypto/rand"
	"crypto/rsa"
	"encoding/binary"
	"fmt"
	"io"
	"io/ioutil"
	"math/big"

	"github.com/golang/glog"
)

type Message struct {
	bytes.Buffer

	xteaEncrypted bool
}

func ReadMessage(r io.Reader) (*Message, error) {
	var len uint16
	if err := binary.Read(r, binary.LittleEndian, &len); err != nil {
		return nil, fmt.Errorf("message len read error: %s", err)
	}

	glog.V(3).Infof("incoming message len: %d", len)

	// TODO(ivucica): implement FastMessage which directly reads from LimitReader avoiding a copy into a bytes.Buffer.
	lr := io.LimitReader(r, int64(len))
	if b, err := ioutil.ReadAll(lr); err != nil {
		return nil, fmt.Errorf("message read error: %s", err)
	} else {
		b := b[4:] // skip checksums. TODO(ivucica): validate checksums
		return &Message{Buffer: *bytes.NewBuffer(b)}, nil
	}
}

func (msg *Message) Read(b []byte) (int, error) {
	n, err := msg.Buffer.Read(b)
	glog.V(3).Infof("read %d bytes", n)
	return n, err
}
func (msg *Message) RSADecryptRemainder(pk *rsa.PrivateKey) error {
	if len(msg.Bytes()) != 128 {
		return fmt.Errorf("rsa encrypted block size = %d; want 128", len(msg.Bytes()))
	}
	/*
		plaintext, err := pk.Decrypt(rand.Reader, msg.Bytes(), nil)
		if err != nil {
			return fmt.Errorf("RSADecryptRemainder: %s (plaintext %s)", err, plaintext)
		}
		msg.Buffer = *bytes.NewBuffer(plaintext)
	*/

	// stolen from DecryptOAEP.
	// This is done because it looks like all public functions in crypto/rsa
	// are performing extra checks.
	c := new(big.Int).SetBytes(msg.Bytes())
	m, err := RSA___decrypt(rand.Reader, pk, c)
	if err != nil {
		return fmt.Errorf("rsa decrypt: %s", err)
	}
	k := 128

	// leftPad returns a new slice of length size. The contents of input are right
	// aligned in the new slice.
	leftPad := func(input []byte, size int) (out []byte) {
		n := len(input)
		if n > size {
			n = size
		}
		out = make([]byte, size)
		copy(out[len(out)-n:], input)
		return
	}

	em := leftPad(m.Bytes(), k)
	msg.Buffer = *bytes.NewBuffer(em)
	return nil

}

func (msg *Message) ReadTibiaString() (string, error) {
	var sz uint16
	err := binary.Read(msg, binary.LittleEndian, &sz)
	if err != nil {
		return "", fmt.Errorf("reading tibia string size: %s", err)
	}
	lr := io.LimitReader(msg, int64(sz))
	b, err := ioutil.ReadAll(lr)
	if err != nil {
		return "", fmt.Errorf("reading tibia string: %s", err)
	}

	return fmt.Sprintf("%s", b), nil
}
