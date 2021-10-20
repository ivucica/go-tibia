// Package full is unstable; it serves as a helper to populate things.Things
// hackily.
package full

import (
	"badc0de.net/pkg/go-tibia/paths"
	"badc0de.net/pkg/go-tibia/things"
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
	// TODO(ivucica): indicate required-nonrequired by something other than empty string, otherwise paths.Find() not finding a file is not an error
	if withSpr {
		return FromPaths(paths.Find("items.otb"), paths.Find("items.xml"), paths.Find("Tibia.dat"), paths.Find("Tibia.spr"))
	} else {
		return FromPaths(paths.Find("items.otb"), paths.Find("items.xml"), paths.Find("Tibia.dat"), "")
	}
}
