package main

import (
	"image"

	"github.com/nfnt/resize"
	"golang.org/x/crypto/ssh/terminal"

	"badc0de.net/pkg/go-tibia/imageprint"
)

func out(img image.Image) {

	if *downsize {
		if w, h, err := terminal.GetSize(0); err == nil { // or int(os.Stdin.Fd())
			img = resize.Thumbnail(uint(w/2), uint(h), img, resize.Lanczos3)
		}
	}

	if *rasterm {
		imageprint.PrintRasTerm(img)
	} else if !*col {
		imageprint.PrintNoColor(img, *blanks)
	} else if *iterm {
		imageprint.PrintITerm(img, "image.png")
	} else if *col256 {
		imageprint.Print256Color(img, *blanks)
	} else {
		imageprint.Print24bit(img, *blanks)
	}
}
