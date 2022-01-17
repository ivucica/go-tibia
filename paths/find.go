package paths

import (
	"github.com/golang/glog"
	"io"
	"os"
)

// Find locates the passed datafile shortname and returns an absolute or
// relative path to find the datafile at.
//
// For example, for "Tibia.pic" it may return
// "mybinary.runfiles/go_tibia/datafiles/Tibia.pic".
//
// TODO(ivucica): Support finding over HTTP.
func Find(fileName string) string {
	possiblePaths := getPossiblePathsImp(fileName)

	for _, path := range possiblePaths {
		if f, err := os.Open(path); err == nil {
			f.Close()
			glog.Infof("paths.Find(%q)=%s", fileName, path)
			return path
		}
	}

	return ""
}

// Open locates the passed file in the same locations that Find would look, and
// opens it. If Find returns an empty string, an error is returned.
//
// TODO(ivucica): Support finding over HTTP.
func Open(fileName string) (interface {
	io.ReadCloser
	io.Seeker
}, error) {
	return openImp(fileName)
}

func NoFindOpen(fileName string) (interface {
	io.ReadCloser
	io.Seeker
}, error) {
	return noFindOpenImp(fileName)
}
