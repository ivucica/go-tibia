package dat

import (
	"os"
	"testing"
)

func TestNewDataset854(t *testing.T) {
	f, err := os.Open("../Tibia.dat")
	if err != nil {
		t.Fatalf("failed to open file: %s", err)
	}
	_, err = NewDataset(f)
	if err != nil {
		t.Fatalf("failed to parse dataset: %s", err)
	}
}
