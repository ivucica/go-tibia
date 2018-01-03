package spr

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"image"
	"image/png"
)

// printImage draws an image using 256color'd ascii art.
func printImage(i image.Image) {
	for y := i.Bounds().Min.Y; y < i.Bounds().Max.Y; y++ {
		for x := i.Bounds().Min.X; x < i.Bounds().Max.X; x++ {
			col := i.At(x, y)
			cR, cG, cB, cA := col.RGBA()
			if cA > 0 {
				fmt.Printf("\x1b[38;2;%d;%d;%dm", cR, cG, cB)
				a := cR + cG + cB
				switch {
				case a < 64:
					fmt.Printf(".")
				case a < 128:
					fmt.Printf("-")
				case a < 192:
					fmt.Printf("=")
				default:
					fmt.Printf("#")
				}
				fmt.Printf("\x1b[0m")
			} else {
				fmt.Printf(" ")
			}
		}
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
