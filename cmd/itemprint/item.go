package main

import (
	"sync"
)

var citemLock sync.Mutex

func citemHandler(idx int, fr, x, y, z int) {

	itm, err := th.ItemWithClientID(uint16(idx), 854)
	if err != nil {
		return
	}

	img := itm.ItemFrame(fr, x, y, z)
	if img == nil {
		return
	}

	out(img)
}

func itemHandler(idx int) {
	itm, err := th.Item(uint16(idx), 854)
	if err != nil {
		return
	}

	img := itm.ItemFrame(0, 0, 0, 0)

	out(img)
}
