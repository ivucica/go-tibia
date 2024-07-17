package itemsotb

import (
	"encoding/xml"
	"io"
)

type xmlItems struct {
	Item []*xmlItem `xml:"item"`
}

type xmlItem struct {
	ID        uint16         `xml:"id,attr,omitempty"`
	Name      string         `xml:"name,attr,omitempty"`
	Article   string         `xml:"article,attr,omitempty"`
	Attribute []xmlAttribute `xml:"attribute,omitempty"`

	Attributes map[string][]string `xml:"-,omitempty"`
}

type xmlAttribute struct {
	Key   string `xml:"key,attr"`
	Value string `xml:"value,attr"`
}

func (otb *Items) AddXMLInfo(r io.Reader) error {
	dec := xml.NewDecoder(r)
	items := xmlItems{}
	if err := dec.Decode(&items); err != nil {
		return err
	}
	for _, it := range items.Item {
		if it.ID >= 20000 {
			// Fluid descriptions. Skip for now.
			continue
		}

		it.Attributes = make(map[string][]string)
		for _, attr := range it.Attribute {
			it.Attributes[attr.Key] = append(it.Attributes[attr.Key], attr.Value)
			if len(it.Attributes[attr.Key]) > 1 {
				// TODO: warn or return error
			}
		}

		item, err := otb.ItemByServerID(it.ID)
		if err != nil {
			return err
		}
		item.xml = it
	}
	return nil
}
