package dat

import (
	"fmt"
	"math"
	"testing"

	"badc0de.net/pkg/go-tibia/paths"
	"badc0de.net/pkg/go-tibia/ttesting"
)

type expectedCounts struct {
	Items           int
	Outfits         int
	Effects         int
	DistanceEffects int

	MaxItemID   int
	MaxOutfitID int
}

func TestDatColor(t *testing.T) {
	c := DatasetColor(156)

	r, g, b, a := c.RGBA()
	if r != 43690 || g != 26214 || b != 0 || a != 65535 {
		t.Errorf("got %d %d %d %d (%g %g %g %g), want %d %d %d %d",
			r, g, b, a,
			float64(r)/math.MaxUint16,
			float64(g)/math.MaxUint16,
			float64(b)/math.MaxUint16,
			float64(a)/math.MaxUint16,
			43690, 26214, 0, 65535)
	}
}

func TestNewDataset(t *testing.T) {
	f, err := paths.Open("Tibia.dat")
	if err != nil {
		t.Fatalf("failed to open file: %s", err)
	}
	ds, err := NewDataset(f)
	if err != nil {
		t.Fatalf("failed to parse dataset: %s", err)
	}

	testableCounts := map[ClientVersion]expectedCounts{
		CLIENT_VERSION_854: {
			Items:           10378,
			Outfits:         351,
			Effects:         69,
			DistanceEffects: 42,

			MaxItemID:   10477,
			MaxOutfitID: 351,
		},
	}
	ver := ds.ClientVersion()

	t.Run(fmt.Sprintf("ver%s", ver.String()), func(t *testing.T) {
		if wantCounts, ok := testableCounts[ver]; ok {
			ttesting.AssertEqualInt(t, fmt.Sprintf("correct item count for minor version %d", ver), len(ds.items), wantCounts.Items)
			ttesting.AssertEqualInt(t, fmt.Sprintf("correct outfit count for minor version %d", ver), len(ds.outfits), wantCounts.Outfits)
			ttesting.AssertEqualInt(t, fmt.Sprintf("correct effect count for minor version %d", ver), len(ds.effects), wantCounts.Effects)
			ttesting.AssertEqualInt(t, fmt.Sprintf("correct distance count for minor version %d", ver), len(ds.distanceEffects), wantCounts.DistanceEffects)

			ttesting.AssertEqualInt(t, fmt.Sprintf("correct max item ID for minor version %d", ver), int(ds.MaxItemID()), wantCounts.MaxItemID)
			ttesting.AssertEqualInt(t, fmt.Sprintf("correct max outfit ID for minor version %d", ver), int(ds.MaxOutfitID()), wantCounts.MaxOutfitID)
		}

		ttesting.AssertEqualInt(t, "first item's ID should be 100", ds.items[0].Id, 100)
		ttesting.AssertEqualInt(t, "last item's ID should be max id", ds.items[len(ds.items)-1].Id, int(ds.Header.ItemCount))
	})

}
