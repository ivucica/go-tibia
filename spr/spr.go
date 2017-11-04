package spr

// This file contains code directly related to decoding the
// spr file format.

import (
	"fmt"
	"image"
	"io"
)

type SpriteSet struct{
	Images []image.Image	
}

func DecodeAll(r io.Reader) (*SpriteSet, error) {
	return nil, fmt.Errorf("not implemented")
}

func (s *SpriteSet) EncodeAll() error {
	return fmt.Errorf("not implemented")
}

