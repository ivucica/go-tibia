//go:build tinygo.wasm
// +build tinygo.wasm

package dom

import (
	"context"
	"fmt"
	"log"

	"badc0de.net/pkg/go-tibia/gameworld"
	"badc0de.net/pkg/go-tibia/things"
)

type jsValue = interface{}

func CompositeMapToDOM(ctx context.Context, window jsValue, m gameworld.MapDataSource, th *things.Things, x, y uint16, floorTop, floorBottom uint8, width, height int, tileW, tileH int) (jsValue, error) {
	log.Println("CompositeMapToDOM not currently implemented in WASI environment")
	return nil, fmt.Errorf("CompositeMapToDOM not currently implemented in WASI environment")
}
