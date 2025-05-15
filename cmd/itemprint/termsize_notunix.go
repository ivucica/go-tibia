//go:build !(aix || darwin || dragonfly || freebsd || linux || netbsd || openbsd || solaris)

package main

import (
	"golang.org/x/crypto/ssh/terminal"
)

type TermSize struct {
	WSRow, WSCol       uint
	WSXPixel, WSYPixel uint
}

func GetTermSize() (TermSize, error) {
	var err error
	var w, h int
	if w, h, err = terminal.GetSize(0); err == nil { // or int(os.Stdin.Fd())
		return TermSize{WSRow: uint(w), WSCol: uint(h)}, nil
	}
	return TermSize{}, err
}
