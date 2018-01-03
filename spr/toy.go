package spr

import (
	"fmt"
	"image"
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
