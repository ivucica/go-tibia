// Package imageprint prints images on terminal. UNSUPPORTED debug package.
//
// This package has an API with no stability guarantees.
package imageprint

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"image"
	ic "image/color"
	"image/png"
	"os"

	"github.com/BourgeoisBear/rasterm"
	"github.com/andybons/gogif"
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

func shade(col ic.Color, escapesTrueColor, blanks, noColor bool) {
	cR, cG, cB, cA := col.RGBA()
	if cA > 0 {
		//fmt.Printf("\x1b[38;5;%dm", a) // TODO(ivucica): Map color to closest entry in xterm 256color palette.
		var d dumper

		if noColor {
			d = &fmtDumper
		} else if escapesTrueColor {
			fmt.Printf("\x1b[48;2;%d;%d;%dm", uint8(cR), uint8(cG), uint8(cB))
			d = &fmtDumper
		} else {
			d = color.RGB(uint8(cR), uint8(cG), uint8(cB), true)
		}
		if blanks {
			d.Printf("  ")
		} else {
			a := ((cR + cG + cB) / 3) >> 8
			switch {
			case a < 32:
				d.Printf("..")
			case a < 64:
				d.Printf("--")
			case a < 128:
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

// Print256Color draws an image using 256color'd ascii art.
func Print256Color(i image.Image, blanks bool) {
	for y := i.Bounds().Min.Y; y < i.Bounds().Max.Y; y++ {
		for x := i.Bounds().Min.X; x < i.Bounds().Max.X; x++ {
			col := i.At(x, y)
			shade(col, false, blanks, false)
		}
		fmt.Printf("\x1b[0m")
		fmt.Printf("\n")
	}
}

// Print24bit draws an image using 24bit color escape sequences by changing background.
func Print24bit(i image.Image, blanks bool) {
	for y := i.Bounds().Min.Y; y < i.Bounds().Max.Y; y++ {
		for x := i.Bounds().Min.X; x < i.Bounds().Max.X; x++ {
			col := i.At(x, y)
			shade(col, true, blanks, false)
		}
		fmt.Printf("\x1b[0m")
		fmt.Printf("\n")
	}
}

// PrintNoColor draws an image without using color escape sequences. Only makes sense with blanks=true.
func PrintNoColor(i image.Image, blanks bool) {
	for y := i.Bounds().Min.Y; y < i.Bounds().Max.Y; y++ {
		for x := i.Bounds().Min.X; x < i.Bounds().Max.X; x++ {
			col := i.At(x, y)
			shade(col, true, blanks, true)
		}
		fmt.Printf("\n")
	}
}

// PrintITerm draws an image using iTerm2's escape sequences.
//
// https://www.iterm2.com/documentation-images.html
func PrintITerm(i image.Image, fn string) {
	if !rasterm.IsTermItermWez() {
		return
	}
	name := base64.StdEncoding.EncodeToString([]byte(fn))
	b := &bytes.Buffer{}
	bEnc := base64.NewEncoder(base64.StdEncoding, b)
	png.Encode(bEnc, i)
	fmt.Printf("\n\033]1337;File=name=%s;inline=1;size=%d,width=%dpx;height=%dpx:%s\a\n", name, len(b.String()), i.Bounds().Size().X, i.Bounds().Size().Y, b.String())
}

// PrintRasTerm draws an image using the RasTerm library.
//
// This should enable drawing in Kitty terminal.
func PrintRasTerm(i image.Image) {
	if rasterm.IsTermKitty() {
		rasterm.Settings{}.KittyWriteImage(os.Stdout, i)
		fmt.Printf("\n")
		return
	}
	if rasterm.IsTermItermWez() {
		rasterm.Settings{}.ItermWriteImage(os.Stdout, i)
		fmt.Printf("\n")
		return
	}
	if capable, err := rasterm.IsSixelCapable(); capable && err == nil {
		palettedImage := image.NewPaletted(i.Bounds(), nil)
		quantizer := gogif.MedianCutQuantizer{NumColor: 64}
		quantizer.Quantize(palettedImage, i.Bounds(), i, image.ZP)

		rasterm.Settings{}.SixelWriteImage(os.Stdout, palettedImage)
		fmt.Printf("\n")
		return
	}
}
