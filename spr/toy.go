package spr

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"image"
	ic "image/color"
	"image/png"

	"github.com/gookit/color"
)

type dumper interface {
	Printf(s string, arg ...interface{})
}
type fmtDumperT struct{}

func (fmtDumperT) Printf(s string, arg ...interface{}) {
	fmt.Printf(s, arg...)
}

var fmtDumper fmtDumperT

func shade(col ic.Color, escapesTrueColor, blanks bool) {
	cR, cG, cB, cA := col.RGBA()
	if cA > 0 {
		//fmt.Printf("\x1b[38;5;%dm", a) // TODO(ivucica): Map color to closest entry in xterm 256color palette.
		var d dumper

		if escapesTrueColor {
			fmt.Printf("\x1b[48;2;%d;%d;%dm", uint8(cR), uint8(cG), uint8(cB))
			d = &fmtDumper
		} else {
			d = color.RGB(uint8(cR), uint8(cG), uint8(cB), true)
		}
		if blanks {
			d.Printf("  ")
		} else {
			a := cR + cG + cB
			switch {
			case a < 64:
				d.Printf("..")
			case a < 128:
				d.Printf("--")
			case a < 192:
				d.Printf("==")
			default:
				d.Printf("##")
			}
		}

		if escapesTrueColor {
			fmt.Printf("\x1b[0m")
		}
	} else {
		fmt.Printf("\x1b[0m  ")
	}
}

// printImage draws an image using 256color'd ascii art.
func printImage256color(i image.Image, blanks bool) {
	for y := i.Bounds().Min.Y; y < i.Bounds().Max.Y; y++ {
		for x := i.Bounds().Min.X; x < i.Bounds().Max.X; x++ {
			col := i.At(x, y)
			shade(col, false, blanks)
		}
		fmt.Printf("\x1b[0m")
		fmt.Printf("\n")
	}
}

// printImage24bit draws an image using 24bit color by changing background.
func printImage24bit(i image.Image, blanks bool) {
	for y := i.Bounds().Min.Y; y < i.Bounds().Max.Y; y++ {
		for x := i.Bounds().Min.X; x < i.Bounds().Max.X; x++ {
			col := i.At(x, y)
			shade(col, true, blanks)
		}
		fmt.Printf("\x1b[0m")
		fmt.Printf("\n")
	}
}

// printImageITerm draws an image using iTerm2's escape sequences.
//
// https://www.iterm2.com/documentation-images.html
func printImageITerm(i image.Image, fn string) {
	name := base64.StdEncoding.EncodeToString([]byte(fn))
	b := &bytes.Buffer{}
	bEnc := base64.NewEncoder(base64.StdEncoding, b)
	png.Encode(bEnc, i)
	fmt.Printf("\n\033]1337;File=name=%s;inline=1;size=%d,width=32px;height=32px:%s\a\n", name, len(b.String()), b.String())
}
