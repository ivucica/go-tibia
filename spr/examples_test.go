package spr

import (
	"fmt"
	"image/png"
	"os"
)

// ExampleDecodeOne decodes a single spr, encodes it into a png, and prints out the image size.
func ExampleDecodeOne() {
	var err error
	f, _ := os.Open(os.Getenv("TEST_SRCDIR") + "/tibia854/Tibia.spr")
	if f == nil {
		f, _ = os.Open("../datafiles/Tibia.spr")
	}
	if f == nil {
		f, _ = os.Open(os.Args[0] + ".runfiles/go_tibia/external/tibia854/Tibia.spr")
	}
	defer f.Close()

	img, err := DecodeOne(f, 423)
	if err != nil {
		fmt.Printf("failed to decode spr: %s", err)
		return
	}
	f, _ = os.Create(os.TempDir() + "/423.png")
	png.Encode(f, img)
	f.Close()

	fmt.Printf("image: %dx%d\n", img.Bounds().Size().X, img.Bounds().Size().Y)
	// Output: image: 32x32
}
