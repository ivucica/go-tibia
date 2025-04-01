package itemsotb

import (
	"fmt"
	"testing"

	"badc0de.net/pkg/go-tibia/paths"
	"badc0de.net/pkg/go-tibia/ttesting"
)

func TestNew(t *testing.T) {
	f, err := paths.Open("items.otb")
	if err != nil {
		t.Skipf("skipping because no file: %v", err)
		return
	}
	otb, err := New(f)
	if err != nil {
		t.Fatalf("failed to parse otb: %s", err)
	}

	ttesting.AssertEqualUint32(t, "correct major version", otb.Version.MajorVersion, 3)
	ttesting.AssertInRangeUint32(t, "correct minor version", uint32(otb.Version.MinorVersion), uint32(CLIENT_VERSION_854), uint32(CLIENT_VERSION_870))
	type expectedCounts struct {
		Items                int
		ClientIDToArrayIndex int
		ServerIDToArrayIndex int

		MinServerID, MaxServerID int
		MinClientID, MaxClientID int
	}
	testableCounts := map[ClientVersion]expectedCounts{
		CLIENT_VERSION_854: expectedCounts{
			Items:                11295,
			ClientIDToArrayIndex: 10378,
			ServerIDToArrayIndex: 11295,

			MinServerID: 100,
			MaxServerID: 11394,
			MinClientID: 100,
			MaxClientID: 10477,
		},
	}
	t.Run(fmt.Sprintf("ver%s", otb.Version.MinorVersion.String()), func(t *testing.T) {
		if wantCounts, ok := testableCounts[otb.Version.MinorVersion]; ok {
			ttesting.AssertEqualInt(t, fmt.Sprintf("correct item count for minor version %d", otb.Version.MinorVersion), len(otb.Items), wantCounts.Items)
			ttesting.AssertEqualInt(t, fmt.Sprintf("correct client id count for minor version %d", otb.Version.MinorVersion), len(otb.ClientIDToArrayIndex), wantCounts.ClientIDToArrayIndex)
			ttesting.AssertEqualInt(t, fmt.Sprintf("correct server id count for minor version %d", otb.Version.MinorVersion), len(otb.ServerIDToArrayIndex), wantCounts.ServerIDToArrayIndex)

			ttesting.AssertEqualInt(t, fmt.Sprintf("correct min server item id for minor version %d", otb.Version.MinorVersion), int(otb.MinServerID), wantCounts.MinServerID)
			ttesting.AssertEqualInt(t, fmt.Sprintf("correct min client item id for minor version %d", otb.Version.MinorVersion), int(otb.MinClientID), wantCounts.MinClientID)
			ttesting.AssertEqualInt(t, fmt.Sprintf("correct max server item id for minor version %d", otb.Version.MinorVersion), int(otb.MaxServerID), wantCounts.MaxServerID)
			ttesting.AssertEqualInt(t, fmt.Sprintf("correct max client item id for minor version %d", otb.Version.MinorVersion), int(otb.MaxClientID), wantCounts.MaxClientID)
		} else {
			t.Errorf("untestable item counts for version %d; please extend the testsuite", otb.Version.MinorVersion)
		}
	})
}
