package spr

// This file contains spr package's functions related to implementing
// image.Image and related interfaces (unless that is better handled
// in spr.go). Anything related to the file actually having multiple
// images encoded is modeled after public interface of the image/gif
// package, and also probably lives in spr.go.

import (
	"fmt"
	"image"
	"image/color"
	"io"

	"encoding/binary"
)

func init() {
	// Spr 8.54: 0x4868ECC9
	image.RegisterFormat("spr", string([]byte{0xC9, 0xEC, 0x68, 0x48}), Decode, DecodeConfig)
	// Pic 8.54: 0x4AE5C3D3
	image.RegisterFormat("pic", string([]byte{0xD3, 0xC3, 0xE5, 0x4A}), Decode, DecodeConfig)
}

type Options struct{}

// DecodeConfig returns the image.Config (width, height, colormodel) for the first image in either spriteset or picture file.
func DecodeConfig(r io.Reader) (image.Config, error) {
	rs, ok := r.(io.ReadSeeker)
	if !ok {
		return image.Config{}, fmt.Errorf("spr: reader must be a ReadSeeker")
	}

	var h Header
	if err := binary.Read(rs, binary.LittleEndian, &h); err != nil {
		return image.Config{}, fmt.Errorf("spr: could not read spr header: %s", err)
	}

	switch h.Signature {
	case 0x4868ECC9: // Spr 8.54
		return image.Config{Width: 32, Height: 32, ColorModel: color.RGBAModel}, nil
	case 0x4AE5C3D3: // Pic 8.54
		// FIXME: implement actual decoding of pic at index 1
		// return image.Config{Width: 640, Height: 480, ColorModel: color.RGBAModel}, nil
		return image.Config{}, fmt.Errorf("DecodeConfig for pic not implemented")
	default: // unknown
		return image.Config{}, fmt.Errorf("not implemented for signature %08x", h.Signature)
	}
}

// DecodeConfigAll is not implemented, but would be useful for PICs.
//
// It would return the dimensions of each of the images in the PIC file, or 32x32 for all images in the SPR file.
func DecodeConfigAll(r io.Reader) ([]image.Config, error) {
	return nil, fmt.Errorf("not implemented")
}

// Decode returns the first image from either spriteset or picture file.
func Decode(r io.Reader) (image.Image, error) {
	rs, ok := r.(io.ReadSeeker)
	if !ok {
		return nil, fmt.Errorf("spr: reader must be a ReadSeeker")
	}

	var h Header
	if err := binary.Read(rs, binary.LittleEndian, &h); err != nil {
		return nil, fmt.Errorf("spr: could not read spr header: %s", err)
	}

	switch h.Signature {
	case 0x4868ECC9: // Spr 8.54
		return decodeOne(rs, h, 1, false)
	case 0x4AE5C3D3: // Pic 8.54
		return decodeOnePic(rs, h, 1)
	default: // unknown
		return nil, fmt.Errorf("not implemented for signature %08x", h.Signature)
	}
}

func Encode(w io.Writer, m image.Image, o *Options) error {
	return fmt.Errorf("not implemented")
}
