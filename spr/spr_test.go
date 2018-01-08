package spr

import (
	"os"
	"testing"

	"image/png"
)

func TestDecodeOne(t *testing.T) {
	f, err := os.Open(os.Getenv("GOPATH") + "/src/badc0de.net/pkg/go-tibia/datafiles/Tibia.spr")
	if err != nil {
		var err2 error
		f, err2 = os.Open(os.Getenv("TEST_SRCDIR") + "/go_tibia/datafiles/Tibia.spr")
		if err2 != nil {
			var err3 error
			f, err3 = os.Open(os.Getenv("TEST_SRCDIR") + "/tibia854/Tibia.spr")
			if err3 != nil {
				var err4 error
				f, err4 = os.Open(os.Args[0] + ".runfiles/go_tibia/external/tibia854/Tibia.spr")
				if err4 != nil {
					t.Fatalf("failed to open file: %s & %s & %s & %s", err, err2, err3, err4)
				}
			}
		}
	}
	defer f.Close()

	img, err := DecodeOne(f, 423) //1231)
	if err != nil {
		t.Fatalf("failed to decode spr: %s", err)
	}
	f2, _ := os.Create("/tmp/bla.png")
	png.Encode(f2, img)
	f2.Close()

	printImage(img)
	printImageITerm(img, "423.png")
}
