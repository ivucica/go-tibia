package compositor

import (
	"testing"

	"badc0de.net/pkg/go-tibia/imageprint"
	"badc0de.net/pkg/go-tibia/things/full"

	"fmt"
	"github.com/nfnt/resize"
	"golang.org/x/crypto/ssh/terminal"
	"image"
	"os"
)

func downsize(_ *testing.T, img image.Image, scale float32) image.Image {
	if w, h, err := terminal.GetSize(0); err == nil { // or int(os.Stdin.Fd())
		return resize.Thumbnail(uint(float32(w/2)*scale), uint(float32(h)*scale), img, resize.Lanczos3)
	} else {
		fmt.Fprintf(os.Stderr, "downsize failed to get terminal size: %v\n", err)
		w = 80
		h = 25
		return resize.Thumbnail(uint(float32(w/2)*scale), uint(float32(h)*scale), img, resize.Lanczos3)
	}
}

func TestCompositeMap(t *testing.T) {
	th, err := full.FromDefaultPaths(true)
	if err != nil {
		t.Fatalf("failed setting up things: %v", err)
	}

	procMDS := gameworld.NewMapDataSource()
	img := CompositeMap(procMDS, th, 100, 100, 7, 7, 18, 16, 32, 32)

	dsimg := downsize(t, img, 1.0)

	imageprint.Print24bit(dsimg, true)
	imageprint.PrintITerm(img, "1.png")
	imageprint.PrintRasTerm(img)
}
