package otbm

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"testing"

	"badc0de.net/pkg/flagutil"
	//"badc0de.net/pkg/go-tibia/ttesting"

	"badc0de.net/pkg/go-tibia/gameworld"
	"badc0de.net/pkg/go-tibia/otb/items"
	"badc0de.net/pkg/go-tibia/paths"
	"badc0de.net/pkg/go-tibia/things"
)

func TestMain(m *testing.M) {
	// make -args -v=2 -logtostderr work.
	flagutil.Parse()

	os.Exit(m.Run())
}

func BenchmarkNew(b *testing.B) {
	th := setupThings(b)

	files := []string{"map.otserv.otbm", "map.Inconcessus-OTBM2JSON.otbm", "map.Inconcessus-OTMapGen.generated.otbm"}
	for _, file := range files {
		b.Run(file, func(b *testing.B) {
			tobNew(b, file, th)
		})
	}
}

func TestNew(t *testing.T) {
	th := setupThings(t)

	files := []string{"map.otserv.otbm", "map.Inconcessus-OTBM2JSON.otbm", "map.Inconcessus-OTMapGen.generated.otbm"}
	for _, file := range files {
		t.Run(file, func(t *testing.T) {
			tobNew(t, file, th)
		})
	}
}

func setupThings(t testOrBenchmark) *things.Things {
	var f, xmlf interface{io.ReadSeeker; io.Closer}
	f, err := paths.Open("items.otb")
	if err != nil {
		t.Fatalf("failed to open items.otb: %s", err)
	}

	xmlf, err = paths.Open("items.xml")
	if err == nil {
		defer xmlf.Close()
	} else {
		t.Fatalf("failed to open items.xml: %s", err)
	}
	//defer f.Close()

	// buffering BEGIN

	// This is an optimization as a lot of performance problems come from too
	// many syscalls for small readers + too many seeks around the file.
	//
	// This optimization should be added to gotserv binary, too, and it should
	// wrap around items.otb loaders too.

	// TODO(ivucica): add error handling here
	// TODO(ivucica): consider migrating otb.New() to use bufio.NewReader() instead, allowing a Peek()/UnreadByte() to replace the use of Seek()
	var sz int64
	fi, err := f.(hasStat).Stat()
	if err != nil {
		t.Errorf("error with stat: %v", err)
	} else {
		sz = fi.Size()
	}

	buf := &bytes.Buffer{}
	buf.Grow(int(sz))
	io.Copy(buf, f) // handle error
	f.Close()
	bufR := bytes.NewReader(buf.Bytes())
	// buffering END

	i, err := itemsotb.New(bufR)
	if err != nil {
		t.Fatalf("failed to load items.otb: %s", err)
	}
	err = i.AddXMLInfo(xmlf)
	if err != nil {
		t.Fatalf("failed to load items.xml: %s", err)
	}

	th, err := things.New()
	if err != nil {
		t.Fatalf("failed to create things registry: %s", err)
	}
	th.AddItemsOTB(i)
	return th
}

type testOrBenchmark interface {
	Fatalf(string, ...interface{})
	Errorf(string, ...interface{})
	Skip(...interface{})
}

type hasStat interface{
	Stat() (os.FileInfo, error)
}

func tobNew(t testOrBenchmark, baseName string, th *things.Things) {
	if baseName == "map.otserv.otbm" && testing.Short() {
		t.Skip("skipping test in short mode")
		return
	}
	f, err := paths.Open(baseName)
	if err != nil {
		t.Fatalf("failed to open file %s: %s", baseName, err)
	}
	//defer f.Close()

	// buffering BEGIN

	// This is an optimization as a lot of performance problems come from too
	// many syscalls for small readers + too many seeks around the file.
	//
	// This optimization should be added to gotserv binary, too, and it should
	// wrap around items.otb loaders too.

	// TODO(ivucica): add error handling here
	// TODO(ivucica): consider migrating otb.New() to use bufio.NewReader() instead, allowing a Peek()/UnreadByte() to replace the use of Seek()

	var sz int64
	fi, err := f.(hasStat).Stat()
	if err != nil {
		t.Errorf("error with stat: %v", err)
	} else {
		sz = fi.Size()
	}

	buf := &bytes.Buffer{}
	buf.Grow(int(sz))
	io.Copy(buf, f) // handle error
	f.Close()
	bufR := bytes.NewReader(buf.Bytes())
	// buffering END

	otbm, err := New(bufR, th)
	if err != nil {
		t.Fatalf("failed to parse otbm %s: %s", baseName, err)
	}

	if tt, ok := t.(*testing.T); ok {
		switch baseName {
		case "map.otserv.otbm":
			temples := []pos{posFromCoord(100, 100, 7), posFromCoord(335, 382, 7)}
			for _, temple := range temples {
				tt.Run(fmt.Sprintf("temple at %v", temple), func(t *testing.T) {

					templeTile, err := otbm.GetMapTile(temple.X(), temple.Y(), temple.Floor())
					if err != nil {
						t.Fatalf("map tile at %v: error: %s", temple, err)
					}
					ground, err := templeTile.GetItem(0)
					if err != nil {
						t.Fatalf("map tile at %v: ground error: %s", temple, err)
					}
					if ground == nil {
						t.Fatalf("map tile at %v: ground is nil", temple)
					}
					if ground.GetServerType() == 0 {
						t.Fatalf("map tile at %v: bad server type 0", temple)
					}
				})
			}

		}
	}

	otbm = otbm
}

/////////////////

func TestMapDescriptionCorrectness(t *testing.T) {
	ds1 := make(chan *Map)
	ds2 := make(chan *Map)
	go func() {
		ds1 <- loadOTBM(t, "range-test-map.otbm")
	}()
	go func() {
		ds2 <- loadOTBM(t, "map.otserv.otbm")
	}()

	gameworld.MapDescriptionEncoding_Test(t, <-ds1, <-ds2)
}

func loadOTBM(t *testing.T, fn string) *Map {
	f, err := paths.Open(fn)
	if err != nil {
		t.Fatalf("failed to open file %q: %s", fn, err)
	}

	th := setupThings(t)

	// buffering BEGIN

	// This is an optimization as a lot of performance problems come from too
	// many syscalls for small readers + too many seeks around the file.
	//
	// This optimization should be added to gotserv binary, too, and it should
	// wrap around items.otb loaders too.

	// TODO(ivucica): add error handling here
	// TODO(ivucica): consider migrating otb.New() to use bufio.NewReader() instead, allowing a Peek()/UnreadByte() to replace the use of Seek()
	var sz int64
	fi, err := f.(hasStat).Stat()
	if err != nil {
		t.Errorf("error with stat: %v", err)
	} else {
		sz = fi.Size()
	}

	buf := &bytes.Buffer{}
	buf.Grow(int(sz))
	io.Copy(buf, f) // handle error
	f.Close()
	bufR := bytes.NewReader(buf.Bytes())
	// buffering END

	otbm, err := New(bufR, th)
	if err != nil {
		t.Fatalf("failed to parse otbm: %s", err)
	}
	return otbm
}
