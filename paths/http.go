package paths

import (
	"bytes"
	"io"
	"log"
	"net/http"
	"os"
	"sync"

	"github.com/pkg/errors"
)

var (
	cache     map[string]*bytes.Buffer
	cacheLock sync.Mutex
)

func getPossiblePathDirsHTTPImp() []string {
	return []string{"http:"}
}

// Find pretends that the file passed is available and returns the local
// filename for the file. In future, it may return a URL.
//
// This is the version intended for use only in WASM environment (i.e.
// inside the browser).
//
// TODO(ivucica): Fetch list of available files and compare to it. Or check with service worker's cache to see if the file is available.
func getPossiblePathsHTTPImp(fileName string) []string {
	return []string{"http:/" + fileName}
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
func openHTTPImp(fileName string) (interface {
	io.ReadCloser
	io.Seeker
}, error) {
	return noFindOpenHTTPImp(fileName)
}

func noFindOpenHTTPImp(fileName string) (interface {
	io.ReadCloser
	io.Seeker
}, error) {
	log.Printf("paths/http.go: NoFindOpen(%q)", fileName)
	cacheLock.Lock()

	if cache == nil {
		log.Printf("paths/http.go: created new http cache")
		cache = make(map[string]*bytes.Buffer)
	}

	log.Printf("paths/http.go: getting http file %q", fileName)
	response, err := http.Get(fileName)
	log.Printf("paths/http.go: http file %q gotten", fileName)
	if err != nil {
		log.Printf("[E] paths/http.go: NoFindOpen(%q): failed to open: %v", fileName, err)
		panic("failed to open " + fileName + ": " + err.Error())
		return nil, errors.Wrapf(err, "go-tibia/paths/NoFindOpen(%q) on wasm: failed to open", fileName)
	}
	if response.StatusCode != http.StatusOK {
		e := os.ErrInvalid
		if response.StatusCode == http.StatusNotFound {
			e = os.ErrNotExist
		}
		log.Printf("[E] paths/http.go: NoFindOpen(%q) on wasm: http response.StatusCode=%v, want 200", fileName, response.StatusCode)
		return nil, errors.Wrapf(e, "go-tibia/paths/NoFindOpen(%q) on wasm: http response.StatusCode=%v, want 200", fileName, response.StatusCode)
	}

	log.Printf("paths/http.go: NoFindOpen(%q): http get complete, locking cache", fileName)

	if buf, ok := cache[fileName]; ok {
		log.Printf("paths/http.go: NoFindOpen(%q): returning reader for cached buffer", fileName)
		cacheLock.Unlock()
		return &bytesReaderWithDummyClose{bytes.NewReader(buf.Bytes())}, nil
	}

	log.Printf("paths/http.go: NoFindOpen(%q): copying from response.Body to buffer", fileName)
	// TODO(ivucica): Explore using ranged reads.
	buf := &bytes.Buffer{}
	if _, err := io.Copy(buf, response.Body); err != nil {
		return nil, errors.Wrap(err, "copying response to seekable buffer")
	}
	response.Body.Close()

	cache[fileName] = buf
	cacheLock.Unlock()

	log.Printf("paths/http.go: NoFindOpen(%q): returning new reader", fileName)
	defer log.Printf("paths/http.go: NoFindOpen(%q): done", fileName)
	return &bytesReaderWithDummyClose{bytes.NewReader(buf.Bytes())}, nil
}

type bytesReaderWithDummyClose struct {
	*bytes.Reader
}

func (bytesReaderWithDummyClose) Close() error {
	return nil
}
