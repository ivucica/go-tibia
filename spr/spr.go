package spr

// This file contains code directly related to decoding the
// spr file format.

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"image"
	"image/color"
	"io"
)

type SpriteSet struct {
	Images []image.Image
}

// DecodeAll decodes all images in the passed reader, and returns a sprite set.
//
// It is currently not implemented.
func DecodeAll(r io.Reader) (*SpriteSet, error) {
	return nil, fmt.Errorf("not implemented")
}

// EncodeAll encodes all images in the sprite set and writes them to the passed writer.
//
// It is currently not implemented.
func (s *SpriteSet) EncodeAll(w io.Writer) error {
	return fmt.Errorf("not implemented")
}

// readerAndByteReader combines io.Reader and io.ByteReader into one interface.
//
// It's necessary for an internal data decoding function.
type readerAndByteReader interface {
	io.Reader
	io.ByteReader
}

type header struct {
	Signature   uint32
	SpriteCount uint16
}

// DecodeOne accepts an io.ReadSeeker positioned at the beginning of a spr-formatted
// file (a sprite set file), finds the image with passed index, and returns the
// requested image as an image.Image.
func DecodeOne(r io.ReadSeeker, which int) (image.Image, error) {
	var h header
	if err := binary.Read(r, binary.LittleEndian, &h); err != nil {
		return nil, fmt.Errorf("could not read spr header: %s", err)
	}
	r.Seek(int64((which-1)*4), io.SeekCurrent) // TODO(ivucica): handle error and return value

	var ptr uint32
	if err := binary.Read(r, binary.LittleEndian, &ptr); err != nil {
		return nil, fmt.Errorf("could not read spr ptr: %s", err)
	}

	r.Seek(int64(ptr), io.SeekStart) // TODO(ivucica): handle error

	return DecodeUpcoming(r)

}

// DecodeUpcoming decodes a single block of spr-format data. This is
// used in both pic and spr files.
func DecodeUpcoming(r io.Reader) (image.Image, error) {
	var colorKey struct{ ColorKeyR, ColorKeyG, ColorKeyB uint8 } // This is colorkey according to http://otfans.net/showpost.php?p=840634&postcount=134. TODO(ivucica): update link as this one is broken.
	if err := binary.Read(r, binary.LittleEndian, &colorKey); err != nil {
		return nil, fmt.Errorf("could not read spr color key: %s", err)
	}

	var size uint16
	if err := binary.Read(r, binary.LittleEndian, &size); err != nil {
		return nil, fmt.Errorf("could not read spr size: %s", err)
	}
	if size > 3444 {
		return nil, fmt.Errorf("spr block too large; got %d, want < 3444", size)
	}

	buf := bytes.Buffer{}
	n, err := buf.ReadFrom(io.LimitReader(r, int64(size)))
	if err != nil {
		return nil, fmt.Errorf("spr block could not be read: %s", err)
	}
	if n != int64(size) {
		return nil, fmt.Errorf("not all of the spr block could be read: read %d, want %d", n, size)
	}

	return decodeData(&buf)
}

func decodeData(r readerAndByteReader) (image.Image, error) {
	img := image.NewRGBA(image.Rect(0, 0, 32, 32))

	transparent := true

	var size uint16
	if err := binary.Read(r, binary.LittleEndian, &size); err != nil {
		return nil, fmt.Errorf("could not read spr segment size: %s", err)
	}
	px := 0
	for {
		if !transparent {
			for i := 0; i < int(size); i++ {
				// TODO(ivucica): handle errors
				cR, _ := r.ReadByte()
				cG, _ := r.ReadByte()
				cB, _ := r.ReadByte()
				col := color.RGBA{
					R: cR,
					G: cG,
					B: cB,
					A: 0xFF,
				}
				img.SetRGBA((px+i)%32, (px+i)/32, col)
			}
		}
		transparent = !transparent

		// next step
		px += int(size)
		if err := binary.Read(r, binary.LittleEndian, &size); err != nil {
			if err != io.EOF {
				return nil, fmt.Errorf("could not read segment size: %s", err)
			} else {
				break
			}

		}
	}
	return img, nil
}
