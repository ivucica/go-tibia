package main

import (
	"fmt"
	"image/color"

	"badc0de.net/pkg/go-tibia/things"
)

func creatureHandler(idx int) {
	cr, err := th.CreatureWithClientID(uint16(idx), 854)
	if err != nil {
		fmt.Printf("err: %v", err)
		return
	}

	img := cr.ColorizedCreatureFrame(0, 2, 0, []color.Color{things.OutfitColor(130), things.OutfitColor(90), things.OutfitColor(25), things.OutfitColor(130)})

	out(img)
}
