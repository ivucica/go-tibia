//go:build !go1.13 || windows
// +build !go1.13 windows

package imageprint

import (
	"flag"
	"fmt"
	"image"
)

var (
	forceITerm = flag.Bool("force_iterm", false, "value to force iterm detection to take (implementation variant: no rasterm)")
)

func isTermItermWez() bool {
	return *forceITerm
}

func PrintRasTerm(i image.Image) {
	fmt.Printf("rasterm not supported below Go 1.13 or on windows\n")
}
