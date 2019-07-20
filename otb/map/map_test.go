package otbm

import (
	"fmt"
	"os"
	"testing"

	"badc0de.net/pkg/flagutil/v1"
	//"badc0de.net/pkg/go-tibia/ttesting"

	"badc0de.net/pkg/go-tibia/otb/items"
	"badc0de.net/pkg/go-tibia/things"
)

func init() {
	// make -args -v=2 -logtostderr work.
	flagutil.Parse()
}

func BenchmarkNew(b *testing.B) {
	th := setupThings(b)

	files := []string{"map.otserv.otbm", "map.Inconcessus-OTBM2JSON.otbm"}
	for _, file := range files {
		b.Run(file, func(b *testing.B) {
			testNew(b, file, th)
		})
	}
}

func TestNew(t *testing.T) {
	th := setupThings(t)

	files := []string{"map.otserv.otbm", "map.Inconcessus-OTBM2JSON.otbm"}
	for _, file := range files {
		t.Run(file, func(t *testing.T) {
			testNew(t, file, th)
		})
	}
}

func setupThings(t testOrBenchmark) *things.Things {
	f, err := os.Open(os.Getenv("GOPATH") + "/src/badc0de.net/pkg/go-tibia/datafiles/items.otb")
	if err != nil {
		var err2 error
		f, err2 = os.Open(os.Getenv("TEST_SRCDIR") + "/go_tibia/datafiles/items.otb")
		if err2 != nil {
			t.Fatalf("failed to open items.otb: %s & %s", err, err2)
		}
	}
	i, err := itemsotb.New(f)
	if err != nil {
		t.Fatalf("failed to load items.otb: %s", err)
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
	Skip(...interface{})
}

func testNew(t testOrBenchmark, baseName string, th *things.Things) {
	if baseName == "map.otserv.otbm" && testing.Short() {
		t.Skip("skipping test in short mode")
		return
	}
	f, err := os.Open(os.Getenv("GOPATH") + "/src/badc0de.net/pkg/go-tibia/datafiles/" + baseName)
	if err != nil {
		var err2 error
		f, err2 = os.Open(os.Getenv("TEST_SRCDIR") + "/go_tibia/datafiles/" + baseName)
		if err2 != nil {
			t.Fatalf("failed to open file %s: %s & %s", baseName, err, err2)
		}
	}
	otbm, err := New(f, th)
	if err != nil {
		t.Fatalf("failed to parse otbm %s: %s", baseName, err)
	}

	if tt := t.(*testing.T); tt != nil {
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
