// Package full is unstable; it serves as a helper to populate things.Things
// hackily.
package full

import (
	"os"

	tdat "badc0de.net/pkg/go-tibia/dat"
	"badc0de.net/pkg/go-tibia/otb/items"
	"badc0de.net/pkg/go-tibia/paths"
	"badc0de.net/pkg/go-tibia/spr"
	"badc0de.net/pkg/go-tibia/things"

	"github.com/pkg/errors"
)

// FromDefaultPaths finds all datafiles supported by things using default
// filepaths as found by the paths package, and adds them to the Things
// structure.
//
// spr can be excluded due to its size.
//
// Appropriate for tests or web frontends. Inappropriate for servers or clients
// where the path should be specifiable by the user on the command line.
func FromDefaultPaths(withSpr bool) (*things.Things, error) {
	t, err := things.New()
	if err != nil {
		return nil, errors.Wrap(err, "creating thing registry")
	}

	f, err := os.Open(paths.Find("items.otb"))
	if err != nil {
		return nil, errors.Wrap(err, "opening items otb file for add")
	}
	itemsOTB, err := itemsotb.New(f)
	f.Close()

	f, err = os.Open(paths.Find("items.xml"))
	if err != nil {
		return nil, errors.Wrap(err, "opening items xml file for add")
	}
	itemsOTB.AddXMLInfo(f)
	f.Close()

	if err != nil {
		return nil, errors.Wrap(err, "parsing items otb for add")
	}
	t.AddItemsOTB(itemsOTB)

	f, err = os.Open(paths.Find("Tibia.dat"))
	if err != nil {
		return nil, errors.Wrap(err, "opening tibia dat file for add")
	}
	dataset, err := tdat.NewDataset(f)
	f.Close()
	if err != nil {
		return nil, errors.Wrap(err, "parsing tibia dat for add")
	}
	t.AddTibiaDataset(dataset)

	if withSpr {
		f, err = os.Open(paths.Find("Tibia.spr"))
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
