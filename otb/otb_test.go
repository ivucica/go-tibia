package otb

import (
	"testing"

	"badc0de.net/pkg/go-tibia/paths"
)

func TestNewOTB(t *testing.T) {
	f, err := paths.Open("items.otb")
	if err != nil {
		t.Fatalf("failed to open file: %s", err)
	}

	_, err = NewOTB(f)
	if err != nil {
		t.Fatalf("failed to parse otb: %s", err)
	}
}
