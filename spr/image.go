package spr

// This file contains spr package's functions related to implementing
// image.Image and related interfaces (unless that is better handled
// in spr.go). Anything related to the file actually having multiple
// images encoded is modeled after public interface of the image/gif
// package, and also probably lives in spr.go.

import (
	"fmt"
	"image"
	"io"
)

func init() {
	// 8.54: 0x4868ECC9
 	image.RegisterFormat("spr", string([]byte{0xC9, 0xEC, 0x68, 0x48}), Decode, DecodeConfig)
}

type Options struct{}


func DecodeConfig(r io.Reader) (image.Config, error) {
	return image.Config{}, fmt.Errorf("not implemented")
}

func Decode(r io.Reader) (image.Image, error) {
	return nil, fmt.Errorf("not implemented")
}

func Encode(w io.Writer, m image.Image, o *Options) error {
	return fmt.Errorf("not implemented")
}
