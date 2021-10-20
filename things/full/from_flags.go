package full

import (
	"badc0de.net/pkg/go-tibia/paths"
	"badc0de.net/pkg/go-tibia/things"
)

var (
	itemsOTBPath string
	itemsXMLPath string
	tibiaDatPath string
	tibiaSprPath string
)

type PathFlag string

const (
	FlagItemsOTBPath = PathFlag("items_otb_path")
	FlagItemsXMLPath = PathFlag("items_xml_path")
	FlagTibiaDatPath = PathFlag("tibia_dat_path")
	FlagTibiaSprPath = PathFlag("tibia_spr_path")
)

// SetupFilePathFlags registers flags to manually define paths to common files
// registerable in things.Things. For example, it will currently register
// --items_otb_path, --items_xml_path, --tibia_dat_path, --tibia_spr_path.
//
// These paths will then be referred to in the FromFilePathFlags function.
func SetupFilePathFlags() {
	paths.SetupFilePathFlag("items.otb", string(FlagItemsOTBPath), &itemsOTBPath)
	paths.SetupFilePathFlag("items.xml", string(FlagItemsXMLPath), &itemsXMLPath)
	paths.SetupFilePathFlag("Tibia.dat", string(FlagTibiaDatPath), &tibiaDatPath)
	paths.SetupFilePathFlag("Tibia.spr", string(FlagTibiaSprPath), &tibiaSprPath)
}

// FromFilePathFlags initializes things.Things populated with files specified by
// the default flags such as --items_otb_path, --items_xml_path etc. The flags
// need to be registered and parsed by the time this function is invoked.
func FromFilePathFlags() (*things.Things, error) {
	return FromPaths(itemsOTBPath, itemsXMLPath, tibiaDatPath, tibiaSprPath)
}

// PathFlagValue returns the value for the passed flag path (such as the path
// to Tibia.dat).
func PathFlagValue(key PathFlag) string {
	switch key {
	case FlagItemsOTBPath:
		return itemsOTBPath
	case FlagItemsXMLPath:
		return itemsXMLPath
	case FlagTibiaDatPath:
		return tibiaDatPath
	case FlagTibiaSprPath:
		return tibiaSprPath
	default:
		return ""
	}
}
