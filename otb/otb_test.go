package otb

import (
	"os"
	"testing"
)

func TestNewOTB(t *testing.T) {
	f, err := os.Open(os.Getenv("GOPATH") + "/src/badc0de.net/pkg/go-tibia/datafiles/items.otb")
	if err != nil {
		var err2 error
		f, err2 = os.Open(os.Getenv("TEST_SRCDIR") + "/go_tibia/external/itemsotb854/file/items.otb")
		if err2 != nil {
			var err3 error
			f, err3 = os.Open(os.Getenv("TEST_SRCDIR") + "/go_tibia/datafiles/items.otb")
			if err3 != nil {
				t.Fatalf("failed to open file: %s & %s & %s", err, err2, err3)
			}
		}
	}
	_, err = NewOTB(f)
	if err != nil {
		t.Fatalf("failed to parse otb: %s", err)
	}
}
