package net

import (
	"bytes"
	"crypto/rand"
	"crypto/rsa"
	"encoding/binary"
	"fmt"
	"hash/adler32"
	"io"
	"io/ioutil"
	"math/big"

	"golang.org/x/crypto/xtea"

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
		b := b[4:] // skip checksums.
		// TODO(ivucica): validate checksums
		return &Message{Buffer: *bytes.NewBuffer(b)}, nil
	}
}

func NewMessage() *Message {
	return &Message{Buffer: bytes.Buffer{}}
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
	// are performing extra checks. And we need just pk.decrypt() which is, as you
	// can see in the name, unexported.
	c := new(big.Int).SetBytes(msg.Bytes())
	m, err := RSA___decrypt(rand.Reader, pk, c)
	if err != nil {
		return fmt.Errorf("rsa decrypt: %s", err)
	}
	k := 128

	// stolen from golang rsa.
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

func (msg *Message) WriteTibiaString(s string) error {
	sz := uint16(len(s))
	err := binary.Write(msg, binary.LittleEndian, sz)
	if err != nil {
		return fmt.Errorf("writing tibia string size: %s", err)
	}

	n, err := msg.WriteString(s)
	if err != nil {
		return fmt.Errorf("writing tibia string: %s", err)
	}

	if n != len(s) {
		return fmt.Errorf("writing tibia string: not all was written")
	}

	return nil
}

// Encrypt reads through the entire message buffer (moving the read cursor),
// and returns a new Message containing the encrypted buffer.
func (msg *Message) Encrypt(xteaKey [16]byte) (*Message, error) {
	glog.V(3).Infoln("input message size: ", msg.Len())
	cipher, err := xtea.NewCipher(xteaKey[:])
	if err != nil {
		return nil, err
	}
	newMsg := NewMessage()
	for msg.Len() > 0 {
		src := [8]byte{}
		msg.Read(src[:]) // TODO(ivucica): handle err. handle n.
		var dst [8]byte

		// The following two lines are disgusting. We are converting not just [8]arrays
		// into []slices. We are also converting 2x uint32 from little endian into big endian.
		// Because this is what Go's XTEA implementation expects within a particular 8-byte
		// block. ¯\_(ツ)_/¯
		cipher.Encrypt(dst[:], []byte{src[3], src[2], src[1], src[0], src[7], src[6], src[5], src[4]})
		newMsg.Write([]byte{dst[3], dst[2], dst[1], dst[0], dst[7], dst[6], dst[5], dst[4]}) // TODO(ivucica): handle err. handle n.
	}
	glog.V(3).Infoln("encrypted message size: ", newMsg.Len())
	return newMsg, nil
}

// Finalize correctly adds the message length and checksum, as well as performs the
// XTEA encryption on the message.
func (msg *Message) Finalize(xteaKey [16]byte) (*Message, error) {

	// add size
	resp, err := msg.finalize(false)
	if err != nil {
		return nil, err
	}

	resp, err = resp.Encrypt(xteaKey)
	if err != nil {
		return nil, err
	}

	// add checksum and size
	resp, err = resp.finalize(true)
	if err != nil {
		return nil, err
	}

	return resp, nil	
}

// finalize prepends the message length and checksum, making it ready for io.Readers
// to read.
//
// TODO(ivucica): We could override the reader for a writable message to first Read()
// out the size and the checksum, thus avoiding the need for a copy.
//
// TODO(ivucica): Maybe we could return the new message.
func (msg *Message) finalize(includeChecksum bool) (*Message, error) {
	newMsg := NewMessage()
	sz := int16(msg.Len())
	origSz := sz
	if includeChecksum {
		sz += 4
	}

	glog.V(2).Infof("finalizing message with size: %d", sz)
	if err := binary.Write(newMsg, binary.LittleEndian, &sz); err != nil {
		return nil, err
	}

	if includeChecksum {
		checksum := adler32.Checksum(msg.Bytes())
		if err := binary.Write(newMsg, binary.LittleEndian, &checksum); err != nil {
			return nil, err
		}
	}

	if written, err := io.Copy(newMsg, msg); err != nil || int16(written) != origSz {
		return nil, fmt.Errorf("Message.Finalize() copy: error %s, written %d/%d", err, written, origSz)
	}

	return newMsg, nil
}
