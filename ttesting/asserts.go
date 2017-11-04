package ttesting

import (
	"testing"
)

func AssertEqualInt(t *testing.T, name string, got, want int) {
	t.Run(name, func(t *testing.T) {
		if got != want {
			t.Errorf("got %d; want %d", got, want)
		}
	})
}

func AssertEqualUint32(t *testing.T, name string, got, want uint32) {
	t.Run(name, func(t *testing.T) {
		if got != want {
			t.Errorf("got %d; want %d", got, want)
		}
	})
}

func AssertInRangeUint32(t *testing.T, name string, got, wantMin, wantMax uint32) {
	t.Run(name, func(t *testing.T) {
		if got < wantMin || got > wantMax {
			t.Errorf("got %d; want [%d,%d]", got, wantMin, wantMax)
		}
	})
}

