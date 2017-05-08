package login

import (
	"encoding/binary"
	gonet "net"

	"badc0de.net/pkg/go-tibia/net"
)

type CharacterListEntry struct {
	CharacterName, CharacterWorld string
	GameFrontend                  gonet.TCPAddr
}

func Error(w *net.Message, errorText string) error {
	if err := w.WriteByte(0x0A); err != nil {
		return err
	}
	if err := w.WriteTibiaString(errorText); err != nil {
		return err
	}
	return nil
}

func FYI(w *net.Message, fyiText string) error {
	if err := w.WriteByte(0x0B); err != nil {
		return err
	}
	if err := w.WriteTibiaString(fyiText); err != nil {
		return err
	}
	return nil
}

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

func ChangeLoginServer(w *net.Message) error {
	// TODO(ivucica): Is this function name correct?
	// This documents 0x28 as 'change session key' with a string
	// http://web.archive.org/web/20170508211101/https://tibiapf.com/showthread.php?59-Tibia-Packets-and-Proxy-Setup&styleid=2
	if err := w.WriteByte(0x28); err != nil {
		return err
	}
	return nil
}

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
