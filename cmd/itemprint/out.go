package main

import (
	"image"

	"github.com/nfnt/resize"

	"badc0de.net/pkg/go-tibia/imageprint"
)

func out(img image.Image) {

	if *downsize {
		termSize, err := GetTermSize()
		if err == nil {
			if (termSize.WSXPixel != 0 && termSize.WSYPixel != 0) && (*rasterm || *iterm) {
				// Prefer printing out in native size if there's a chance we print out an image rather than pixels.
				//
				// Ideally this can only be decided when either rasterm or iterm renderers perform the print, but this hack might help anyway until the whole of imageprint is refactored and moved into a different package.
				img = resize.Thumbnail(termSize.WSXPixel/2, termSize.WSYPixel/2, img, resize.Lanczos3)
			} else {
				img = resize.Thumbnail(termSize.WSRow/2, termSize.WSCol/2, img, resize.Lanczos3)
			}
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
