package itemsotb

import (
	"fmt"
	"os"
	"testing"

	"badc0de.net/pkg/go-tibia/v1/ttesting"
)

func TestNew(t *testing.T) {
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
	otb, err := New(f)
	if err != nil {
		t.Fatalf("failed to parse otb: %s", err)
	}

	ttesting.AssertEqualUint32(t, "correct major version", otb.Version.MajorVersion, 3)
	ttesting.AssertInRangeUint32(t, "correct minor version", otb.Version.MinorVersion, CLIENT_VERSION_854, CLIENT_VERSION_870)
	type expectedCounts struct {
		Items                int
		ClientIDToArrayIndex int
		ServerIDToArrayIndex int
	}
	testableCounts := map[uint32]expectedCounts{
		CLIENT_VERSION_854: expectedCounts{
			Items:                11295,
			ClientIDToArrayIndex: 10378,
			ServerIDToArrayIndex: 11295,
		},
	}
	if wantCounts, ok := testableCounts[otb.Version.MinorVersion]; ok {
		ttesting.AssertEqualInt(t, fmt.Sprintf("correct item count for minor version %d", otb.Version.MinorVersion), len(otb.Items), wantCounts.Items)
		ttesting.AssertEqualInt(t, fmt.Sprintf("correct client id count for minor version %d", otb.Version.MinorVersion), len(otb.ClientIDToArrayIndex), wantCounts.ClientIDToArrayIndex)
		ttesting.AssertEqualInt(t, fmt.Sprintf("correct server id count for minor version %d", otb.Version.MinorVersion), len(otb.ServerIDToArrayIndex), wantCounts.ServerIDToArrayIndex)
	} else {
		t.Errorf("untestable item counts for version %d; please extend the testsuite", otb.Version.MinorVersion)
	}
}
