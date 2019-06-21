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

func TestNew(t *testing.T) {
	f, err := os.Open(os.Getenv("GOPATH") + "/src/badc0de.net/pkg/go-tibia/datafiles/map.otbm")
	if err != nil {
		var err2 error
		f, err2 = os.Open(os.Getenv("TEST_SRCDIR") + "/go_tibia/datafiles/map.otbm")
		if err2 != nil {
			t.Fatalf("failed to open file: %s & %s", err, err2)
		}
	}
	otbm, err := New(f)
	if err != nil {
		t.Fatalf("failed to parse otbm: %s", err)
	}

	otbm = otbm
}
