// Package xmls is unstable. It contains functionality for reading some of the OpenTibia Server's XML data files besides just items.xml, but there are no guarantees on its name or stability.
package xmls

import (
	"encoding/xml"
	"io"
	"strings"
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

// n.b. this belongs not in outfit definition but in creature definition
//
// (and by creature, this means NPCs and enemies -- so not even in Things)
func (c *OutfitListEntry) Temp__ExternalLink() string {
	if c.Name == "" || c.Name == "unnamed creature" {
		return ""
	}
	return "https://tibia.fandom.com/wiki/" + strings.Replace(strings.Replace(strings.Title(c.Name), " ", "_", -1), " Of ", " of ", -1)
}

// n.b. this belongs not in outfit definition but in creature definition
//
// (and by creature, this means NPCs and enemies -- so not even in Things)
func (c *OutfitListEntry) Temp__ExternalLootStatsLink() string {
	if c.Name == "" || c.Name == "unnamed creature" {
		return ""
	}
	return "https://tibia.fandom.com/index.php?title=Loot_Statistics:" + strings.Replace(strings.Replace(strings.Title(c.Name), " ", "_", -1), " Of ", " of ", -1) + "&action=raw"
}

func ReadOutfits(r io.Reader) (Outfits, error) {
	dec := xml.NewDecoder(r)
	outfits := Outfits{}
	if err := dec.Decode(&outfits); err != nil {
		return outfits, err
	}
	return outfits, nil
}
