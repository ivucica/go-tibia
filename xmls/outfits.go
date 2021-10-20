// Package xmls is unstable. It contains functionality for reading some of the OpenTibia Server's XML data files besides just items.xml, but there are no guarantees on its name or stability.
package xmls

import (
	"encoding/xml"
	"io"
)

type Outfits struct {
	xml.Name `xml:"outfits"`
	Outfit   []Outfit `xml:"outfit"`
}

type Outfit struct {
	ID      int               `xml:"id,attr"`
	Premium int               `xml:"premium,attr"`
	Default string            `xml:"default,attr"`
	List    []OutfitListEntry `xml:"list"`
}

type OutfitType string

const (
	OutfitTypeMale   = OutfitType("male")
	OutfitTypeFemale = OutfitType("female")
)

type OutfitListEntry struct {
	Type     OutfitType `xml:"type,attr"`
	LookType int        `xml:"looktype,attr"`
	Name     string     `xml:"name,attr"`
}

func ReadOutfits(r io.Reader) (Outfits, error) {
	dec := xml.NewDecoder(r)
	outfits := Outfits{}
	if err := dec.Decode(&outfits); err != nil {
		return outfits, err
	}
	return outfits, nil
}
