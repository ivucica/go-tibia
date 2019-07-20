package things

import (
	"badc0de.net/pkg/go-tibia/dat"
	"badc0de.net/pkg/go-tibia/otb/items"
	"badc0de.net/pkg/go-tibia/spr"
)

type Things struct {
	items     *itemsotb.Items
	dataset   *dat.Dataset
	spriteSet *spr.SpriteSet
}

func New() (*Things, error) {
	return &Things{}, nil
}

func (t *Things) AddItemsOTB(i *itemsotb.Items) error {
	t.items = i
	return nil
}

func (t *Things) AddTibiaDataset(d *dat.Dataset) error {
	t.dataset = d
	return nil
}

func (t *Things) AddSpriteSet(s *spr.SpriteSet) error {
	t.spriteSet = s
	return nil
}

func (t *Things) Temp__GetClientIDForServerID(serverID uint16, clientVersion uint16) uint16 {
	itm, err := t.items.ItemByServerID(serverID)
	if err != nil {
		return 0
	}
	if attr, ok := itm.Attributes[itemsotb.ITEM_ATTR_CLIENTID]; ok {
		return attr.(uint16)
	} else {
		return 0
	}
}

func (t *Things) Temp__GetItemFromOTB(serverID uint16, clientVersion uint16) *itemsotb.Item {
	itm, err := t.items.ItemByServerID(serverID)
	if err != nil {
		return nil
	}
	return itm
}
