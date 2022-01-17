package full

import (
	"log" // TODO: remove std log

	tdat "badc0de.net/pkg/go-tibia/dat"
	"badc0de.net/pkg/go-tibia/otb/items"
	"badc0de.net/pkg/go-tibia/spr"
	"badc0de.net/pkg/go-tibia/things"
	"badc0de.net/pkg/go-tibia/paths"

	"github.com/pkg/errors"
)

// FromPaths populates a things.Things datastructure using datafiles found
// at passed paths. Any path passed as an empty string will be omitted.
func FromPaths(itemsOTBPath, itemsXMLPath, tibiaDatPath, tibiaSprPath string) (*things.Things, error) {
	// TODO(ivucica): indicate required-nonrequired by something other than empty string, otherwise paths.Find() not finding a file is not an error
	t, err := things.New()
	if err != nil {
		return nil, errors.Wrap(err, "creating thing registry")
	}

	if itemsOTBPath != "" {
		log.Printf("full.FromPaths(): opening items otb: %q", itemsOTBPath)
		f, err := paths.NoFindOpen(itemsOTBPath)
		if err != nil {
			return nil, errors.Wrap(err, "opening items otb file for add")
		}
		itemsOTB, err := itemsotb.New(f)
		f.Close()
		if err != nil {
			return nil, errors.Wrap(err, "parsing items otb for add")
		}
		t.AddItemsOTB(itemsOTB)

		if itemsXMLPath != "" {
		log.Printf("full.FromPaths(): opening items xml: %q", itemsXMLPath)
			f, err := paths.NoFindOpen(itemsXMLPath)
			if err != nil {
				return nil, errors.Wrap(err, "opening items xml file for add")
			}
			itemsOTB.AddXMLInfo(f)
			f.Close()
		}
	}

	if tibiaDatPath != "" {
		log.Printf("full.FromPaths(): opening tibia dat: %q", tibiaDatPath)
		f, err := paths.NoFindOpen(tibiaDatPath)
		if err != nil {
			return nil, errors.Wrap(err, "opening tibia dat file for add")
		}
		dataset, err := tdat.NewDataset(f)
		f.Close()
		if err != nil {
			return nil, errors.Wrap(err, "parsing tibia dat for add")
		}
		t.AddTibiaDataset(dataset)
	}

	if tibiaSprPath != "" {
		log.Printf("full.FromPaths(): opening tibia spr: %q", tibiaSprPath)
		f, err := paths.NoFindOpen(tibiaSprPath)
		if err != nil {
			return nil, errors.Wrap(err, "opening tibia spr file for add")
		}
		spriteset, err := spr.DecodeAll(f)
		f.Close()
		if err != nil {
			return nil, errors.Wrap(err, "parsing tibia spr for add")
		}
		t.AddSpriteSet(spriteset)
	}

	return t, nil
}
