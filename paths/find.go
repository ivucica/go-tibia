package paths

import (
	"os"
)

// Find locates the passed datafile shortname and returns an absolute or
// relative path to find the datafile at.
//
// For example, for "Tibia.pic" it may return
// "mybinary.runfiles/go_tibia/datafiles/Tibia.pic".
func Find(fileName string) string {
	possiblePaths := []string{
		os.Getenv("GOPATH") + "/src/badc0de.net/pkg/go-tibia/datafiles/" + fileName,
		os.Args[0] + ".runfiles/go_tibia/datafiles/" + fileName,
		os.Args[0] + ".runfiles/go_tibia/external/itemsotb854/file/" + fileName,
		os.Args[0] + ".runfiles/go_tibia/external/tibia854/" + fileName,
		"datafiles/" + fileName,
		fileName,
	}
	if os.Getenv("TEST_SRCDIR") != "" {
		possiblePaths = append(possiblePaths,
			os.Getenv("TEST_SRCDIR")+"/go_tibia/datafiles/"+fileName,
			os.Getenv("TEST_SRCDIR")+"/go_tibia/external/itemsotb854/file/"+fileName,
			os.Getenv("TEST_SRCDIR")+"/go_tibia/external/tibia854/"+fileName)
	}

	for _, path := range possiblePaths {
		if f, err := os.Open(path); err == nil {
			f.Close()
			return path
		}
	}

	return ""
}

// Open locates the passed file in the same locations that Find would look, and
// opens it. If Find returns an empty string, an error is returned.
func Open(fileName string) (*os.File, error) {
	path := Find(fileName)
	if path == "" {
		return nil, os.ErrNotExist
	}
	return os.Open(path)
}
