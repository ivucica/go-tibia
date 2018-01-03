package login

// This file contains various functions to transmit a particular message to the client
// as well as some model structures.

import (
	"encoding/binary"
	gonet "net"

	"badc0de.net/pkg/go-tibia/net"
)

// CharacterListEntry represents a single character presented on the character list.
type CharacterListEntry struct {
	CharacterName, CharacterWorld string
	GameFrontend                  gonet.TCPAddr
}

// Error writes a login error network message to the passed net.Message.
//
// Passed error text will be included.
func Error(w *net.Message, errorText string) error {
	if err := w.WriteByte(0x0A); err != nil {
		return err
	}
	if err := w.WriteTibiaString(errorText); err != nil {
		return err
	}
	return nil
}

// FYI writes an FYI network message to the passed net.Messsage.
//
// Passed FYI text will be included.
func FYI(w *net.Message, fyiText string) error {
	if err := w.WriteByte(0x0B); err != nil {
		return err
	}
	if err := w.WriteTibiaString(fyiText); err != nil {
		return err
	}
	return nil
}

// MOTD writes the message-of-the-day network message to the passed net.Message.
//
// Passed motdText will be included.
//
// The motdText should begin with ascii-encoded decimal number identifying the
// sequence number of the MOTD, then it should be followed by a newline
// set of characters (ascii 13+10, \r\n). The number is used by the client
// to avoid bothering the user with the same message that was already seen.
func MOTD(w *net.Message, motdText string) error {
	if err := w.WriteByte(0x0B); err != nil {
		return err
	}
	if err := w.WriteTibiaString(motdText); err != nil {
		return err
	}
	return nil
}

// Patching functions 0x1E-0x20 not supported.

// ChangeLoginServer sends network message with ID 0x28, of uncertain functionality.
func ChangeLoginServer(w *net.Message) error {
	// TODO(ivucica): Is this function name correct?
	// This documents 0x28 as 'change session key' with a string
	// http://web.archive.org/web/20170508211101/https://tibiapf.com/showthread.php?59-Tibia-Packets-and-Proxy-Setup&styleid=2
	if err := w.WriteByte(0x28); err != nil {
		return err
	}
	return nil
}

// CharacterList sends the network message containing the passed characters
// on the character list, and tells the user they have premiumDays left.
func CharacterList(w *net.Message, chars []CharacterListEntry, premiumDays uint16) error {
	if err := w.WriteByte(0x64); err != nil {
		return err
	}

	if err := w.WriteByte(byte(len(chars))); err != nil {
		return err
	}

	for _, char := range chars {
		w.WriteTibiaString(char.CharacterName)
		w.WriteTibiaString(char.CharacterWorld)
		w.Write(char.GameFrontend.IP.To4())
		port := uint16(char.GameFrontend.Port)
		binary.Write(w, binary.LittleEndian, &port)
	}

	binary.Write(w, binary.LittleEndian, &premiumDays)
	// New versions might include 32bit premiumTimestamp

	return nil
}

// TODO(ivucica): support 0xA0 (160) "client corrupted" receiving string
// TODO(ivucica): support 0x81 (129), unknown and possibly receives nothing
// TODO(ivucica): support 0x11 "update" receiving string
