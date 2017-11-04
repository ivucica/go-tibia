package dat

import (
	"fmt"
	"os"
	"testing"

	"badc0de.net/pkg/go-tibia/ttesting"
)

type expectedCounts struct {
	Items           int
	Outfits         int
	Effects         int
	DistanceEffects int
}

func TestNewDataset(t *testing.T) {
	f, err := os.Open(os.Getenv("GOPATH") + "/src/badc0de.net/pkg/go-tibia/datafiles/Tibia.dat")
	if err != nil {
		var err2 error
		f, err2 = os.Open(os.Getenv("TEST_SRCDIR") + "/go_tibia/datafiles/Tibia.dat")
		if err2 != nil {
			var err3 error
			f, err3 = os.Open(os.Getenv("TEST_SRCDIR") + "/tibia854/Tibia.dat")
			if err3 != nil {
				t.Fatalf("failed to open file: %s & %s & %s", err, err2, err3)
			}
		}
	}
	ds, err := NewDataset(f)
	if err != nil {
		t.Fatalf("failed to parse dataset: %s", err)
	}

	testableCounts := map[int]expectedCounts{
		CLIENT_VERSION_854: expectedCounts{
			Items:           10378,
			Outfits:         351,
			Effects:         69,
			DistanceEffects: 42,
		},
	}
	ver := ds.ClientVersion()

	if wantCounts, ok := testableCounts[ver]; ok {
		if len(ds.items) != wantCounts.Items {
			ttesting.AssertEqualInt(t, fmt.Sprintf("correct item count for minor version %d", ver), len(ds.items), wantCounts.Items)
		}
		if len(ds.outfits) != wantCounts.Outfits {
			ttesting.AssertEqualInt(t, fmt.Sprintf("correct outfit count for minor version %d", ver), len(ds.outfits), wantCounts.Outfits)
		}
		if len(ds.effects) != wantCounts.Effects {
			ttesting.AssertEqualInt(t, fmt.Sprintf("correct effect count for minor version %d", ver), len(ds.effects), wantCounts.Effects)
		}
		if len(ds.distanceEffects) != wantCounts.DistanceEffects {
			ttesting.AssertEqualInt(t, fmt.Sprintf("correct distance count for minor version %d", ver), len(ds.distanceEffects), wantCounts.DistanceEffects)
		}
	}

	ttesting.AssertEqualInt(t, "first item's ID should be 100", ds.items[0].Id, 100)
	ttesting.AssertEqualInt(t, "last item's ID should be max id", ds.items[len(ds.items)-1].Id, int(ds.header.ItemCount))


}
