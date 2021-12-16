package main

import (
	"os"

	"github.com/golang/glog"

	"badc0de.net/pkg/go-tibia/spr"
)

func picOpen() (ReadSeekerCloser, error) {
	return os.Open(tibiaPicPath)
}

func picHandler(idx int) {
	f, err := picOpen()
	if err != nil {
		return
	}
	defer f.Close()

	img, err := spr.DecodeOnePic(f, idx)
	if err != nil {
		glog.Errorf("error decoding spr: %v", err)
		return
	}

	out(img)
}
