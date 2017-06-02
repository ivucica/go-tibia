package dat

import (
	"encoding/binary"
	"io"
)

type Dataset struct {
	items           []Item
	outfits         []Outfit
	effects         []Effect
	distanceEffects []DistanceEffect
}

type header struct {
	Signature                                                uint32
	ItemCount, OutfitCount, EffectCount, DistanceEffectCount uint16
}

type Item struct{}
type Outfit struct{}
type Effect struct{}
type DistanceEffect struct{}

func NewDataset(r io.Reader) (*Dataset, error) {
	h := header{}
	binary.Read(r, binary.LittleEndian, &h)

	dataset := Dataset{
		items:           make([]Item, h.ItemCount),
		outfits:         make([]Outfit, h.OutfitCount),
		effects:         make([]Effect, h.EffectCount),
		distanceEffects: make([]DistanceEffect, h.DistanceEffectCount),
	}
	return &dataset, nil
}
