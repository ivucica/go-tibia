package spr

import (
	"fmt"
	"image/png"
	"os"

	"badc0de.net/pkg/go-tibia/paths"
)

// ExampleDecodeOne decodes a single spr, encodes it into a png, and prints out the image size.
func ExampleDecodeOne() {
	var err error
	f, err := paths.Open("Tibia.spr")
	if err != nil {
		panic(err.Error())
	}
	defer f.Close()

	img, err := DecodeOne(f, 423)
	if err != nil {
		fmt.Printf("failed to decode spr: %s", err)
		return
	}
	pngf, err := os.Create(os.TempDir() + "/423.png")
	if err != nil {
		fmt.Printf("failed to open %q for write", os.TempDir()+"/423.png")
		return
	}
	png.Encode(pngf, img)
	f.Close()

	fmt.Printf("image: %dx%d\n", img.Bounds().Size().X, img.Bounds().Size().Y)
	// Output: image: 32x32
}
