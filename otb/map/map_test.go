package otbm

import (
	//"fmt"
	"os"
	"testing"

	"badc0de.net/pkg/flagutil/v1"
	//"badc0de.net/pkg/go-tibia/ttesting"
)

func init() {
	// make -args -v=2 -logtostderr work.
	flagutil.Parse()
}

func BenchmarkNew(b *testing.B) {
	files := []string{"map.otbm.otserv", "map.otbm.shdpl-libotbm"}
	for _, file := range files {
		b.Run(file, func(b *testing.B) {
			testNew(b, file)
		})
	}
}

func TestNew(t *testing.T) {
	files := []string{"map.otbm.otserv", "map.otbm.shdpl-libotbm"}
	for _, file := range files {
		t.Run(file, func(t *testing.T) {
			testNew(t, file)
		})
	}
}

type testOrBenchmark interface {
	Fatalf(string, ...interface{})
	Skip(...interface{})
}

func testNew(t testOrBenchmark, baseName string) {
	if baseName == "map.otbm.otserv" && testing.Short() {
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
	otbm, err := New(f)
	if err != nil {
		t.Fatalf("failed to parse otbm %s: %s", baseName, err)
	}

	otbm = otbm
}


