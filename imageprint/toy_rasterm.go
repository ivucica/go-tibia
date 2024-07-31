//go:build go1.13 && !windows
// +build go1.13,!windows

package imageprint

import (
	"flag"
	"fmt"
	"image"
	"os"

	"github.com/BourgeoisBear/rasterm"
	"github.com/andybons/gogif"
)

var (
	// TODO: remove this flag and just don't use it if --iterm=true is explicitly set
	forceITerm = flag.Bool("force_iterm", false, "disable iterm detection, force returning true (implementation variant: has rasterm)")
)

func isTermItermWez() bool {
	if *forceITerm {
		return true
	}
	return rasterm.IsTermItermWez()
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
