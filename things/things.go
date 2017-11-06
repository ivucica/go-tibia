package items

import (
	"badc0de.net/pkg/go-tibia/otb/items"
	"badc0de.net/pkg/go-tibia/spr"
)

type Itemset struct {
	Items     map[uint16]Item
	SpriteSet *spr.SpriteSet
}

type Item struct {
	Graphics []int
}

func NewItemset() (*Itemset, error) {
	return &Itemset{}, nil
}

func (*Itemset) AddFromItemsOTB(*itemsotb.Items) error {
	// not implemented
	return nil
}
