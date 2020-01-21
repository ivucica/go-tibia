package main

import (
	"flag"
	"fmt"
	"image"
	"image/draw"
	"image/png"
	"io"
	"net/http"
	"os"
	"strconv"
	"sync"

	"badc0de.net/pkg/flagutil/v1"

	tdat "badc0de.net/pkg/go-tibia/dat"
	"badc0de.net/pkg/go-tibia/otb/items"
	"badc0de.net/pkg/go-tibia/spr"
	"badc0de.net/pkg/go-tibia/things"

	"github.com/golang/glog"
	"github.com/gorilla/mux"
)

var (
	listenAddress = flag.String("listen_address", ":8080", "http listen address for gotweb")

	itemsOTBPath string
	itemsXMLPath string
	tibiaDatPath string
	tibiaSprPath string
	tibiaPicPath string
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

func picOpen() (ReadSeekerCloser, error) {
	return os.Open(tibiaPicPath)
}

func setupFilePathFlags() {
	setupFilePathFlag("items.otb", "items_otb_path", &itemsOTBPath)
	setupFilePathFlag("items.xml", "items_xml_path", &itemsXMLPath)
	setupFilePathFlag("Tibia.dat", "tibia_dat_path", &tibiaDatPath)
	setupFilePathFlag("Tibia.spr", "tibia_spr_path", &tibiaSprPath)
	setupFilePathFlag("Tibia.pic", "tibia_pic_path", &tibiaPicPath)
}

func setupFilePathFlag(fileName, flagName string, flagPtr *string) {
	possiblePaths := []string{
		os.Getenv("GOPATH") + "/src/badc0de.net/pkg/go-tibia/datafiles/" + fileName,
		os.Args[0] + ".runfiles/go_tibia/datafiles/" + fileName,
		os.Args[0] + ".runfiles/go_tibia/external/itemsotb854/file/" + fileName,
		os.Args[0] + ".runfiles/go_tibia/external/tibia854/" + fileName,
	}

	didReg := false
	for _, path := range possiblePaths {
		if f, err := os.Open(path); err == nil {
			f.Close()
			flag.StringVar(flagPtr, flagName, path, "Path to "+fileName)
			didReg = true
			break
		}
	}
	if !didReg {
		flag.StringVar(flagPtr, flagName, "", "Path to "+fileName)
	}
}

func thingsOpen() *things.Things {
	t, err := things.New()
	if err != nil {
		glog.Errorln("creating thing registry", err)
		return nil
	}

	f, err := os.Open(itemsOTBPath)
	if err != nil {
		glog.Errorln("opening items otb file for add", err)
		return nil
	}
	itemsOTB, err := itemsotb.New(f)
	f.Close()

	f, err = os.Open(itemsXMLPath)
	if err != nil {
		glog.Errorln("opening items xml file for add", err)
		return nil
	}
	itemsOTB.AddXMLInfo(f)
	f.Close()

	if err != nil {
		glog.Errorln("parsing items otb for add", err)
		return nil
	}
	t.AddItemsOTB(itemsOTB)

	f, err = os.Open(tibiaDatPath)
	if err != nil {
		glog.Errorln("opening tibia dat file for add", err)
		return nil
	}
	dataset, err := tdat.NewDataset(f)
	f.Close()
	if err != nil {
		glog.Errorln("parsing tibia dat for add", err)
		return nil
	}
	t.AddTibiaDataset(dataset)

	f, err = os.Open(tibiaSprPath)
	if err != nil {
		glog.Errorln("opening tibia spr file for add", err)
		return nil
	}
	spriteset, err := spr.DecodeAll(f)
	f.Close()
	if err != nil {
		glog.Errorln("parsing tibia spr for add", err)
		return nil
	}
	t.AddSpriteSet(spriteset)

	return t
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
		glog.Errorf("error decoding spr: %v", err)
		return
	}
	w.Header().Set("Content-Type", "image/png")
	w.WriteHeader(http.StatusOK)
	png.Encode(w, img)
}

func picHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	idx, err := strconv.Atoi(vars["idx"])
	if err != nil {
		http.Error(w, "idx not a number", http.StatusBadRequest)
		return
	}

	f, err := picOpen()
	if err != nil {
		http.Error(w, "failed to open data file", http.StatusNotFound)
		return
	}
	defer f.Close()

	img, err := spr.DecodeOnePic(f, idx)
	if err != nil {
		http.Error(w, "failed to decode pic", http.StatusInternalServerError)
		glog.Errorf("error decoding pic: %v", err)
		return
	}

	var src image.Rectangle
	var dst image.Rectangle
	if x := r.URL.Query().Get("x"); x != "" {
		src.Min.X, _ = strconv.Atoi(x)
	}
	if y := r.URL.Query().Get("y"); y != "" {
		src.Min.Y, _ = strconv.Atoi(y)
	}
	if w := r.URL.Query().Get("w"); w != "" {
		dst.Max.X, _ = strconv.Atoi(w)
		src.Max.X = src.Min.X + dst.Max.X

	}
	if h := r.URL.Query().Get("h"); h != "" {
		dst.Max.Y, _ = strconv.Atoi(h)
		src.Max.Y = src.Min.Y + dst.Max.Y
	}

	if dst.Max.X != 0 && dst.Max.Y != 0 {
		oldImg := img
		img = image.NewRGBA(dst)
		draw.Draw(img.(draw.Image), dst, oldImg, src.Min, draw.Over)
	}

	w.Header().Set("Content-Type", "image/png")
	w.WriteHeader(http.StatusOK)
	png.Encode(w, img)
}

var (
	th *things.Things
)

func itemHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	idx, err := strconv.Atoi(vars["idx"])
	if err != nil {
		http.Error(w, "idx not a number", http.StatusBadRequest)
		return
	}

	itm, err := th.Item(uint16(idx), 854)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	img := itm.ItemFrame(0, 0, 0, 0)

	w.Header().Set("Content-Type", "image/png")
	w.WriteHeader(http.StatusOK)
	png.Encode(w, img)
}

var citemLock sync.Mutex

func citemHandler(w http.ResponseWriter, r *http.Request) {
	citemLock.Lock()
	defer citemLock.Unlock()

	vars := mux.Vars(r)
	idx, err := strconv.Atoi(vars["idx"])
	if err != nil {
		http.Error(w, "idx not a number", http.StatusBadRequest)
		return
	}

	var p struct{ x, y, z, fr int }
	if x := r.URL.Query().Get("x"); x != "" {
		p.x, _ = strconv.Atoi(x)
		// ignore invalid x
	}
	if y := r.URL.Query().Get("y"); y != "" {
		p.y, _ = strconv.Atoi(y)
		// ignore invalid y
	}
	if z := r.URL.Query().Get("z"); z != "" {
		p.z, _ = strconv.Atoi(z)
		// ignore invalid z
	}
	if fr := r.URL.Query().Get("fr"); fr != "" {
		p.fr, _ = strconv.Atoi(fr)
		// ignore invalid fr
	}

	itm, err := th.ItemWithClientID(uint16(idx), 854)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	img := itm.ItemFrame(p.fr, p.x, p.y, p.z)
	if img == nil {
		http.Error(w, "bad image", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "image/png")
	w.WriteHeader(http.StatusOK)
	png.Encode(w, img)
}

func main() {
	setupFilePathFlags()
	flagutil.Parse()

	th = thingsOpen()

	r := mux.NewRouter()
	r.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		pg := 0
		pgSize := 50

		if pgStr := r.URL.Query().Get("page"); pgStr != "" {
			if pgConv, err := strconv.Atoi(pgStr); err == nil {
				pg = pgConv - 1
			}
		}

		w.Header().Set("Content-Type", "text/html")

		itemCIDMin := 100
		itemCIDMax := 10477 // TODO: ask things.Things
		pgMin := 0
		pgMax := (itemCIDMax - itemCIDMin) / pgSize

		if pg < pgMin {
			pg = pgMin
		}
		if pg > pgMax {
			pg = pgMax
		}

		for i := pgMin; i < pgMax+1; i++ {
			if i == pg {
				fmt.Fprintf(w, `<b><a href="?page=%d">%d</a></b> `, i+1, i+1)
			} else {
				fmt.Fprintf(w, `<a href="?page=%d">%d</a> `, i+1, i+1)
			}
		}
		fmt.Fprintf(w, "\n")

		fmt.Fprintf(w, "<ul>")
		for i := 100 + pg*pgSize; i < 100+pg*pgSize+pgSize; i++ {
			var name string
			wid := 32
			hei := 32
			if itm, err := th.ItemWithClientID(uint16(i), 854); err == nil {
				name = fmt.Sprintf("%d: %s", i, itm.Name())
				sz := itm.GraphicsSize()
				wid = sz.W
				hei = sz.H
			} else {
				name = fmt.Sprintf("%d", i)
			}
			fmt.Fprintf(w, "<li><dt>%s</dt><dd><img width=%d height=%d src=/item/c%d></dd>\n", name, wid, hei, i)
		}
	})
	r.HandleFunc("/spr/{idx:[0-9]+}", sprHandler)
	r.HandleFunc("/item/{idx:[0-9]+}", itemHandler)
	r.HandleFunc("/item/c{idx:[0-9]+}", citemHandler)
	r.HandleFunc("/pic/{idx:[0-9]+}", picHandler)

	glog.Infof("beginning to serve")
	glog.Fatal(http.ListenAndServe(*listenAddress, r))
}
