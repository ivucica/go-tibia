//go:build !js && !wasm
// +build !js,!wasm

package paths

import (
	"io"
)

func getPossiblePathDirsImp() []string {
	return getPossiblePathDirsFSImp()
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
	return getPossiblePathsFSImp(fileName)
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
	return openFSImp(fileName)
}

func noFindOpenImp(fileName string) (interface {
	io.ReadCloser
	io.Seeker
}, error) {
	return noFindOpenFSImp(fileName)
}
