package otb

import (
	"fmt"
	"os"
	"testing"
)

func assertEqualInt(t *testing.T, name string, got, want int) {
	t.Run(name, func(t *testing.T) {
		if got != want {
			t.Errorf("got %d; want %d", got, want)
		}
	})
}

func assertEqualUint32(t *testing.T, name string, got, want uint32) {
	t.Run(name, func(t *testing.T) {
		if got != want {
			t.Errorf("got %d; want %d", got, want)
		}
	})
}

func assertInRangeUint32(t *testing.T, name string, got, wantMin, wantMax uint32) {
	t.Run(name, func(t *testing.T) {
		if got < wantMin || got > wantMax {
			t.Errorf("got %d; want [%d,%d]", got, wantMin, wantMax)
		}
	})
}

func TestNewItemsOTB(t *testing.T) {
	f, err := os.Open("../items.otb")
	if err != nil {
		t.Fatalf("failed to open file: %s", err)
	}
	otb, err := NewItemsOTB(f)
	if err != nil {
		t.Fatalf("failed to parse otb: %s", err)
	}

	assertEqualUint32(t, "correct major version", otb.Version.MajorVersion, 3)
	assertInRangeUint32(t, "correct minor version", otb.Version.MinorVersion, CLIENT_VERSION_854, CLIENT_VERSION_870)
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
		assertEqualInt(t, fmt.Sprintf("correct item count for minor version %d", otb.Version.MinorVersion), len(otb.Items), wantCounts.Items)
		assertEqualInt(t, fmt.Sprintf("correct client id count for minor version %d", otb.Version.MinorVersion), len(otb.ClientIDToArrayIndex), wantCounts.ClientIDToArrayIndex)
		assertEqualInt(t, fmt.Sprintf("correct server id count for minor version %d", otb.Version.MinorVersion), len(otb.ServerIDToArrayIndex), wantCounts.ServerIDToArrayIndex)
	} else {
		t.Errorf("untestable item counts for version %d; please extend the testsuite", otb.Version.MinorVersion)
	}
}
