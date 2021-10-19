package spr

import (
	"fmt"
	"image"
	"os"
	"testing"

	"badc0de.net/pkg/go-tibia/imageprint"
	"badc0de.net/pkg/go-tibia/paths"
	"github.com/nfnt/resize"
	"golang.org/x/crypto/ssh/terminal"
)

func TestDecodeOne(t *testing.T) {
	f, err := paths.Open("Tibia.spr")
	if err != nil {
		t.Fatalf("failed to open file: %s", err)
	}
	defer f.Close()

	img, err := DecodeOne(f, 423) // 1231)
	if err != nil {
		t.Fatalf("failed to decode spr: %s", err)
	}
	// f2, _ := os.Create("/tmp/423.png")
	// png.Encode(f2, img)
	// f2.Close()

	imageprint.Print256Color(img, true)
	imageprint.Print24bit(img, true)
	imageprint.PrintITerm(img, "423.png")
	imageprint.PrintRasTerm(img)
}

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

func TestDecodeOnePic(t *testing.T) {
	f, err := paths.Open("Tibia.pic")
	if err != nil {
		t.Fatalf("failed to open file: %s", err)
	}
	defer f.Close()

	img, err := DecodeOnePic(f, 1)
	if err != nil {
		t.Fatalf("failed to decode spr: %s", err)
	}

	dsimg := downsize(t, img, 0.25)

	imageprint.Print256Color(dsimg, true)
	imageprint.Print24bit(dsimg, true)
	imageprint.PrintITerm(img, "1.png")
	imageprint.PrintRasTerm(img)
}
