//go:build tinygo.wasm

package main

import (
	"log"
)

func (*console) Write(p []byte) (n int, err error) {
	log.Println(p)
	return len(p), nil
}
