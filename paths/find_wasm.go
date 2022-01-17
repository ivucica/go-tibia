// +build js,wasm

package paths

import (
	"io"
	"log"
	"strings"
)

func getPossiblePathDirsImp() []string {
	return append(getPossiblePathDirsFSImp(), getPossiblePathDirsHTTPImp()...)
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
	return append(getPossiblePathsFSImp(fileName), getPossiblePathsHTTPImp(fileName)...)
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
	paths := getPossiblePathsImp(fileName)

	for _, f := range paths {
		log.Printf("testing path %s", f)
		if strings.HasPrefix(f, "http://") {
			o, err := noFindOpenHTTPImp(f)
			if err == nil {
				return o, nil
			}
		} else if strings.HasPrefix(f, "http:/") {
			// relative http path (shouldn't begin with / but eh)
			o, err := openHTTPImp(f[len("http:/"):])
			if err == nil {
				return o, nil
			}
		} else {
			o, err := noFindOpenFSImp(f)
			if err == nil {
				return o, nil
			}
		}
	}

	// bleh, try once again with fs imp just to get the error.
	// this whole thing needs to be cleaned up anyway.
	return openFSImp(fileName)
	return openHTTPImp(fileName)
}

func noFindOpenImp(fileName string) (interface {
	io.ReadCloser
	io.Seeker
}, error) {
	if strings.HasPrefix(fileName, "http://") {
		log.Printf("wasm noFindOpenIimp: using nofindopen http imp: %v", fileName)
		return noFindOpenHTTPImp(fileName)
	} else if strings.HasPrefix(fileName, "http:/") {
		log.Printf("wasm noFindOpenIimp: using nofindopen http imp 2: %v -> %v", fileName, fileName[len("http:/"):])
		return noFindOpenHTTPImp(fileName[len("http:/"):])
	}
	log.Printf("wasm noFindOpenIimp: using nofindopen fs imp: %v", fileName)
	return noFindOpenFSImp(fileName)
}
