// +build !js,!wasm

package paths

import (
	"flag"
	"github.com/golang/glog"
	"io"
	"os"
)

func getPossiblePathDirsImp() []string {
	var possiblePaths []string

	possiblePaths = append(possiblePaths,
		os.Getenv("GOPATH")+"/src/badc0de.net/pkg/go-tibia/datafiles",
		os.Args[0]+".runfiles/go_tibia/datafiles",
		os.Args[0]+".runfiles/go_tibia/external/itemsotb854/file",
		os.Args[0]+".runfiles/go_tibia/external/tibia854",
		"datafiles",
		".",
	)
	if os.Getenv("TEST_SRCDIR") != "" {
		possiblePaths = append(possiblePaths,
			os.Getenv("TEST_SRCDIR")+"/go_tibia/datafiles",
			os.Getenv("TEST_SRCDIR")+"/go_tibia/external/itemsotb854/file",
			os.Getenv("TEST_SRCDIR")+"/go_tibia/external/tibia854")
	}

	return possiblePaths
}

// getPossiblePathsImp locates the passed datafile shortname and returns an
// absolute or relative path to find the datafile at.
//
// For example, for "Tibia.pic" it may return
// "mybinary.runfiles/go_tibia/datafiles/Tibia.pic".
//
// Used in Find(). This is the local filesystem, native binary implementation.
//
// TODO(ivucica): Support finding over HTTP.
func getPossiblePathsImp(fileName string) []string {
	var possiblePaths []string

	flagsForFilesLock.Lock()
	if fl, ok := flagsForFiles[fileName]; ok {
		f := flag.Lookup(fl)
		if f != nil && f.Value != nil && f.Value.String() != "" {
			possiblePaths = append([]string{f.Value.String()}, possiblePaths...)
			glog.Infof("found flag --%s containing data for file %s (%s)", fl, fileName, possiblePaths[0])
		} else if f != nil && f.DefValue != "" {
			possiblePaths = append([]string{f.DefValue}, possiblePaths...)
			glog.Infof("found flag --%s containing data for file %s (%s)", fl, fileName, possiblePaths[0])
		}

	}
	flagsForFilesLock.Unlock()

	for _, pp := range getPossiblePathDirsImp() {
		possiblePaths = append(possiblePaths, pp+"/"+fileName)
	}

	return possiblePaths
}

// openImp locates the passed file in the same locations that Find would look,
// and opens it. If Find returns an empty string, an error is returned.
//
// This is the local filesystem, native binary implementation.
//
// TODO(ivucica): Support finding over HTTP.
func openImp(fileName string) (interface {
	io.ReadCloser
	io.Seeker
}, error) {
	path := Find(fileName)
	if path == "" {
		return nil, os.ErrNotExist
	}
	glog.Infof("paths.Open(%q) redirected to %q", fileName, path)
	f, err := NoFindOpen(path)
	if err != nil {
		glog.Errorf("paths.Open(%q) error %v: %v", fileName, path, err)
	}
	return f, err
}

func noFindOpenImp(fileName string) (interface {
	io.ReadCloser
	io.Seeker
}, error) {
	return os.Open(fileName)
}
