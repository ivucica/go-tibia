// +build js,wasm

package paths

import (
	"bytes"
	"io"
	"net/http"
	"os"

	"github.com/pkg/errors"
)

// Find pretends that the file passed is available and returns the local
// filename for the file. In future, it may return a URL.
//
// This is the version intended for use only in WASM environment (i.e.
// inside the browser).
//
// TODO(ivucica): Fetch list of available files and compare to it. Or check with service worker's cache to see if the file is available.
func Find(fileName string) string {
	return fileName
}

// Open would usually locate the passed file in the same locations that Find
// would look in, and open it. If Find were to return an empty string, an
// error would be is returned.
//
// But the file is just opened over HTTP, given the environment we're in (WASM)
// and how we expect worker to just work.
//
// TODO(ivucica): Support finding the file in the 'best available' place by
// actually implementing Find.
func Open(fileName string) (interface{io.ReadCloser; io.Seeker}, error) {
	response, err := http.Get("Tibia.spr")
	if err != nil {
		return nil, errors.Wrapf(err, "go-tibia/paths/Open(%q) on wasm: failed to open", fileName)
	}
	if response.StatusCode != http.StatusOK {
		e := os.ErrInvalid
		if response.StatusCode == http.StatusNotFound {
			e = os.ErrNotExist
		}
		return nil, errors.Wrapf(e, "go-tibia/paths/Open(%q) on wasm: http response.StatusCode=%v, want 200", fileName, response.StatusCode)
	}

	// TODO(ivucica): Explore using ranged reads.
	buf := &bytes.Buffer{}
	if _, err := io.Copy(buf, response.Body); err != nil {
		return nil, errors.Wrap(err, "copying response to seekable buffer")
	}
	response.Body.Close()

	return &bytesReaderWithDummyClose{bytes.NewReader(buf.Bytes())}, nil
}

type bytesReaderWithDummyClose struct {
	*bytes.Reader
}

func (bytesReaderWithDummyClose) Close() error {
	return nil
}
