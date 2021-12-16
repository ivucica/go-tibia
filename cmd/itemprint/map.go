package main

import (
	"os"

	"github.com/golang/glog"
	"github.com/pkg/errors"

	"badc0de.net/pkg/go-tibia/gameworld" // for map compositor
	"badc0de.net/pkg/go-tibia/otb/map"   // for map loader
)

func mapHandler(mapPath string, x, y, w, h, bot, top int) {
	m, err := loadMap(mapPath)
	if err != nil {
		glog.Errorf("error loading map: %v", err)
		return
	}

	// TODO: more input validation! never allow for number inside CompositeMap to go negative, e.g.
	img := gameworld.CompositeMap(m, th, uint16(x), uint16(y), uint8(top), uint8(bot), w, h, 32, 32)

	out(img)

}

func loadMap(mapPath string) (gameworld.MapDataSource, error) {
	var m gameworld.MapDataSource
	if mapPath == ":test:" {
		m = gameworld.NewMapDataSource()
	} else {
		f, err := os.Open(mapPath)
		if err != nil {
			return nil, errors.Wrap(err, "opening map file")
		}
		m, err = otbm.New(f, th)
		if err != nil {
			return nil, errors.Wrap(err, "reading map file")
		}
		f.Close()
	}
	return m, nil
}
