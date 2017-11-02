package otb

import (
	"os"
	"testing"
)

func TestNewItemsOTB(t *testing.T) {
	f, err := os.Open("../items.otb")
	if err != nil {
		t.Fatalf("failed to open file: %s", err)
	}
	_, err = NewItemsOTB(f)
	if err != nil {
		t.Fatalf("failed to parse otb: %s", err)
	}
}
