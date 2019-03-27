package main

import (
	"flag"
	"fmt"
	"image/png"
	"io"
	"net/http"
	"os"
	"strconv"

	"badc0de.net/pkg/flagutil/v1"

	"badc0de.net/pkg/go-tibia/spr"

	"github.com/golang/glog"
	"github.com/gorilla/mux"
)

var (
	listenAddress = flag.String("listen_address", ":8080", "http listen address for gotweb")
)

type ReadSeekerCloser interface {
	io.Reader
	io.Seeker
	io.Closer
}

func sprOpen() (ReadSeekerCloser, error) {
	f, err := os.Open(os.Getenv("GOPATH") + "/src/badc0de.net/pkg/go-tibia/datafiles/Tibia.spr")
	if err != nil {
		var err2 error
		f, err2 = os.Open(os.Getenv("TEST_SRCDIR") + "/go_tibia/datafiles/Tibia.spr")
		if err2 != nil {
			var err3 error
			f, err3 = os.Open(os.Getenv("TEST_SRCDIR") + "/tibia854/Tibia.spr") // TODO: do we want to hardcode 854?
			if err3 != nil {
				var err4 error
				f, err4 = os.Open(os.Args[0] + ".runfiles/go_tibia/external/tibia854/Tibia.spr")
				if err4 != nil {
					return nil, fmt.Errorf("could not open spr") // TODO: replace with err, err2, err3 + err4?
				}
			}
		}
	}
	return f, nil
}

func sprHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	idx, err := strconv.Atoi(vars["idx"])
	if err != nil {
		http.Error(w, "idx not a number", http.StatusBadRequest)
		return
	}

	f, err := sprOpen()
	if err != nil {
		http.Error(w, "failed to open data file", http.StatusNotFound)
		return
	}
	defer f.Close()

	img, err := spr.DecodeOne(f, idx)
	if err != nil {
		http.Error(w, "failed to decode spr", http.StatusInternalServerError)
	}
	w.Header().Set("Content-Type", "image/png")
	w.WriteHeader(http.StatusOK)
	png.Encode(w, img)
}

func main() {
	flagutil.Parse()

	r := mux.NewRouter()
	r.HandleFunc("/spr/{idx:[0-9]+}", sprHandler)

	glog.Fatal(http.ListenAndServe(*listenAddress, r))
}
