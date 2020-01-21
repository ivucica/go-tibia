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
	"sync"

	"github.com/golang/glog"
)

type SpriteSet struct {
	//Images []image.Image

	buf *bytes.Reader
	m   sync.Mutex
}

func (s *SpriteSet) Image(idx int) image.Image {
	s.m.Lock()
	s.buf.Seek(0, io.SeekStart)
	spr, err := DecodeOne(s.buf, idx)
	if err != nil {
		glog.Errorf("error decoding sprite %d : %v", idx, err)
		s.m.Unlock()
		return nil
	}
	s.m.Unlock()
	return spr
}

type ColorKey struct{ ColorKeyR, ColorKeyG, ColorKeyB uint8 }

// DecodeAll decodes all images in the passed reader, and returns a sprite set.
//
// It is currently implemented as an in-memory buffer which can be queried to
// return a particular sprite.
func DecodeAll(r io.Reader) (*SpriteSet, error) {
	buf := &bytes.Buffer{}
	io.Copy(buf, r)

	ss := &SpriteSet{
		buf: bytes.NewReader(buf.Bytes()),
	}

	return ss, nil
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

type picHeader struct {
	Width, Height uint8
	ColorKey      ColorKey
}

// DecodeOne accepts an io.ReadSeeker positioned at the beginning of a spr-formatted
// file (a sprite set file), finds the image with passed index, and returns the
// requested image as an image.Image.
func DecodeOne(r io.ReadSeeker, which int) (image.Image, error) {
	return decodeOne(r, which, false)
}

func decodeOne(r io.ReadSeeker, which int, isPic bool) (image.Image, error) {
	if which == 0 {
		return nil, fmt.Errorf("not found")
	}

	var h header
	if err := binary.Read(r, binary.LittleEndian, &h); err != nil {
		return nil, fmt.Errorf("could not read spr header: %s", err)
	}
	if which >= int(h.SpriteCount) {
		return nil, fmt.Errorf("not found")
	}

	var width int
	var height int
	var ptrs []uint32
	if !isPic {
		width = 1
		height = 1
		r.Seek(int64((which-1)*4), io.SeekCurrent) // TODO(ivucica): handle error and return value
	} else {
		for i := 0; i < which; i++ {
			var ph picHeader
			if err := binary.Read(r, binary.LittleEndian, &ph); err != nil {
				return nil, fmt.Errorf("could not read pic header: %s", err)
			}
			width = int(ph.Width)
			height = int(ph.Height)
			if i != which-1 {
				r.Seek(int64(width)*int64(height)*4, io.SeekCurrent) // TODO(ivucica): handle error and return value
			}
		}
	}
	ptrs = make([]uint32, width*height)

	if err := binary.Read(r, binary.LittleEndian, &ptrs); err != nil {
		return nil, fmt.Errorf("could not read ptrs: %s", err)
	}

	img := image.NewRGBA(image.Rect(0, 0, 32*width, 32*height))
	for x := 0; x < width; x++ {
		for y := 0; y < height; y++ {
			ptr := ptrs[y*width+x]
			r.Seek(int64(ptr), io.SeekStart) // TODO(ivucica): handle error

			if !isPic {
				var colorKey ColorKey // This is colorkey according to http://otfans.net/showpost.php?p=840634&postcount=134. TODO(ivucica): update link as this one is broken.
				if err := binary.Read(r, binary.LittleEndian, &colorKey); err != nil {
					return nil, fmt.Errorf("could not read spr color key: %s", err)
				}
			}
			if err := decodeUpcoming(r, img, x*32, y*32); err != nil {
				return nil, err
			}
		}
	}
	return img, nil
}

// DecodeOnePic behaves like DecodeOne, except it accepts .pic formatted files.
func DecodeOnePic(r io.ReadSeeker, which int) (image.Image, error) {
	return decodeOne(r, which, true)
}

// DecodeUpcoming decodes a single block of spr-format data. This is
// used in both pic and spr files.
func DecodeUpcoming(r io.Reader) (image.Image, error) {
	i := image.NewRGBA(image.Rect(0, 0, 32, 32))
	if err := decodeUpcoming(r, i, 0, 0); err != nil {
		return nil, err
	}
	return i, nil
}
func decodeUpcoming(r io.Reader, img *image.RGBA, x, y int) error {
	var size uint16
	if err := binary.Read(r, binary.LittleEndian, &size); err != nil {
		return fmt.Errorf("could not read spr size: %s", err)
	}
	if size > 3444 {
		return fmt.Errorf("spr block too large; got %d, want < 3444", size)
	}

	if size == 0 {
		return nil
	}

	buf := bytes.Buffer{}
	n, err := buf.ReadFrom(io.LimitReader(r, int64(size)))
	if err != nil {
		return fmt.Errorf("spr block could not be read: %s", err)
	}
	if n != int64(size) {
		return fmt.Errorf("not all of the spr block could be read: read %d, want %d", n, size)
	}

	return decodeData(&buf, img, x, y)
}

func decodeData(r readerAndByteReader, img *image.RGBA, x, y int) error {
	transparent := true

	var size uint16
	if err := binary.Read(r, binary.LittleEndian, &size); err != nil {
		return fmt.Errorf("could not read spr segment size: %s", err)
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
				img.SetRGBA(x+(px+i)%32, y+(px+i)/32, col)
			}
		}
		transparent = !transparent

		// next step
		px += int(size)
		if err := binary.Read(r, binary.LittleEndian, &size); err != nil {
			if err != io.EOF {
				return fmt.Errorf("could not read segment size: %s", err)
			} else {
				break
			}

		}
	}
	return nil
}
