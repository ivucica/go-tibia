package main

import (
	"os"

	"github.com/golang/glog"

	"badc0de.net/pkg/go-tibia/spr"
)

func sprOpen() (ReadSeekerCloser, error) {
	return os.Open(tibiaSprPath)
}

func sprHandler(idx int) {
	f, err := sprOpen()
	if err != nil {
		return
	}
	defer f.Close()

	img, err := spr.DecodeOne(f, idx)
	if err != nil {
		glog.Errorf("error decoding spr: %v", err)
		return
	}

	out(img)
}
