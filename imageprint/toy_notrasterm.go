// +build !go1.13

package imageprint

import (
	"fmt"
	"image"
)

func isTermItermWez() bool {
	return false
}

func PrintRasTerm(i image.Image) {
	fmt.Printf("rasterm not supported below Go 1.13\n")
}
