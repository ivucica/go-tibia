package main

import (
	"encoding/binary"
	"html/template"
	_ "net/http/pprof" // Default mux should not be served publicly; it's actually hidden behind a flag.

	"flag"
	"fmt"
	"image"
	"image/color"
	"image/color/palette"
	"image/draw"
	"image/gif"
	"image/jpeg"
	"image/png"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"sync"

	"badc0de.net/pkg/flagutil"

	tdat "badc0de.net/pkg/go-tibia/dat"
	"badc0de.net/pkg/go-tibia/gameworld" // for map compositor
	"badc0de.net/pkg/go-tibia/otb/items"
	"badc0de.net/pkg/go-tibia/otb/map" // for map loader
	"badc0de.net/pkg/go-tibia/paths"
	"badc0de.net/pkg/go-tibia/spr"
	"badc0de.net/pkg/go-tibia/things"

	"github.com/bradfitz/iter"
	"github.com/golang/glog"
	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
)

var (
	listenAddress      = flag.String("listen_address", ":8080", "http listen address for gotweb")
	debugListenAddress = flag.String("debug_listen_address", "", "http listen address for pprof (and other stuff registered on default serve mux)")

	itemsOTBPath string
	itemsXMLPath string
	tibiaDatPath string
	tibiaSprPath string
	tibiaPicPath string
	mapPath      string
	htmlPath     string
)

type ReadSeekerCloser interface {
	io.Reader
	io.Seeker
	io.Closer
}

func sprOpen() (ReadSeekerCloser, error) {
	return os.Open(tibiaSprPath)
}

func picOpen() (ReadSeekerCloser, error) {
	return os.Open(tibiaPicPath)
}

func setupFilePathFlags() {
	paths.SetupFilePathFlag("items.otb", "items_otb_path", &itemsOTBPath)
	paths.SetupFilePathFlag("items.xml", "items_xml_path", &itemsXMLPath)
	paths.SetupFilePathFlag("Tibia.dat", "tibia_dat_path", &tibiaDatPath)
	paths.SetupFilePathFlag("Tibia.spr", "tibia_spr_path", &tibiaSprPath)
	paths.SetupFilePathFlag("Tibia.pic", "tibia_pic_path", &tibiaPicPath)
	paths.SetupFilePathFlag("map.otbm", "map_path", &mapPath)
	paths.SetupFilePathFlag("html/index.html", "index_html_path", &htmlPath)
	htmlPath = filepath.Dir(htmlPath)
}

func thingsOpen() *things.Things {
	// TODO(ivucica): Add functionality to things/full package to support flags-based loading.
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

	f.Seek(0, io.SeekStart)
	var h spr.Header
	binary.Read(f, binary.LittleEndian, &h)

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

	generation := 1 // bump if the way we generate it changes
	mime := "image/png"
	etag := fmt.Sprintf(`W/"20211019:pic:%d:%08x:%d:%d.%d.%d.%d.%s"`, generation, h.Signature, idx, src.Min.X, src.Min.Y, src.Max.X, src.Max.Y, mime)
	if r.Header.Get("If-None-Match") == etag {
		w.Header().Set("Cache-Control", "public; max-age=36000") // 36000 = 10h
		w.Header().Set("ETag", etag)
		w.WriteHeader(http.StatusNotModified)
		return
	}

	if dst.Max.X != 0 && dst.Max.Y != 0 {
		oldImg := img
		img = image.NewRGBA(dst)
		draw.Draw(img.(draw.Image), dst, oldImg, src.Min, draw.Over)
	}

	w.Header().Set("Content-Type", "image/png")
	w.Header().Set("Cache-Control", "public; max-age=36000") // 36000 = 10h
	w.Header().Set("ETag", etag)
	if s, err := os.Stat(tibiaPicPath); err == nil {
		w.Header().Set("Last-Modified", s.ModTime().Format(http.TimeFormat))
	}
	w.WriteHeader(http.StatusOK)
	png.Encode(w, img)
}

var (
	th           *things.Things
	itemLock     sync.Mutex
	creatureLock sync.Mutex
)

func itemHandler(w http.ResponseWriter, r *http.Request) {
	itemLock.Lock()
	defer itemLock.Unlock()

	vars := mux.Vars(r)
	idx, err := strconv.Atoi(vars["idx"])
	if err != nil {
		http.Error(w, "idx not a number", http.StatusBadRequest)
		return
	}

	generation := 1 // bump if the way we generate it changes
	mime := "image/png"
	// TODO: if we support x, y, z etc this should be supported in etag too.
	etag := fmt.Sprintf(`W/"item:%d:%08x:%08x:%d:%s"`, generation, th.SpriteSetSignature(), th.TibiaDatasetSignature(), idx, mime)

	if r.Header.Get("If-None-Match") == etag {
		w.Header().Set("Cache-Control", "public; max-age=36000") // 36000 = 10h
		w.Header().Set("ETag", etag)
		w.WriteHeader(http.StatusNotModified)
		return
	}

	itm, err := th.Item(uint16(idx), 854)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	img := itm.ItemFrame(0, 0, 0, 0)
	if img == nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", mime)
	w.Header().Set("Cache-Control", "public; max-age=36000") // 36000 = 10h
	w.Header().Set("ETag", etag)
	if s, err := os.Stat(tibiaSprPath); err == nil {
		// TODO: max of tibia.dat, tibia.spr, maybe more
		w.Header().Set("Last-Modified", s.ModTime().Format(http.TimeFormat))
	}
	w.WriteHeader(http.StatusOK)
	png.Encode(w, img)
}

func citemHandler(w http.ResponseWriter, r *http.Request) {
	itemLock.Lock()
	defer itemLock.Unlock()

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

	generation := 1 // bump if the way we generate it changes
	mime := "image/png"
	etag := fmt.Sprintf(`W/"20211019:item:%d:%08x:%08x:%d:%d.%d.%d.%d.%s"`, generation, th.SpriteSetSignature(), th.TibiaDatasetSignature(), idx, p.fr, p.x, p.y, p.z, mime)
	if r.Header.Get("If-None-Match") == etag {
		w.Header().Set("Cache-Control", "public; max-age=36000") // 36000 = 10h
		w.Header().Set("ETag", etag)
		w.WriteHeader(http.StatusNotModified)
		return
	}

	img := itm.ItemFrame(p.fr, p.x, p.y, p.z)
	if img == nil {
		http.Error(w, "bad image", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", mime)
	w.Header().Set("Cache-Control", "public; max-age=3600")
	w.Header().Set("ETag", etag)
	if s, err := os.Stat(tibiaSprPath); err == nil {
		// TODO: max of tibia.dat, tibia.spr, maybe more
		w.Header().Set("Last-Modified", s.ModTime().Format(http.TimeFormat))
	}
	w.WriteHeader(http.StatusOK)
	png.Encode(w, img)
}

func creatureHandler(w http.ResponseWriter, r *http.Request) {
	creatureLock.Lock()
	defer creatureLock.Unlock()

	vars := mux.Vars(r)
	idx, err := strconv.Atoi(vars["idx"])
	if err != nil {
		http.Error(w, "idx not a number", http.StatusBadRequest)
		return
	}

	dir, err := strconv.Atoi(vars["dir"])
	if err != nil {
		http.Error(w, "dir not a number", http.StatusBadRequest)
		return
	}

	fr, err := strconv.Atoi(vars["fr"])
	if err != nil {
		http.Error(w, "fr not a number", http.StatusBadRequest)
		return
	}

	cr, err := th.CreatureWithClientID(uint16(idx), 854)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	var p struct {
		outfitOverlayMask things.OutfitOverlayMask
		col               [4]color.Color
	}
	p.col[0] = things.OutfitColor(0)
	p.col[1] = things.OutfitColor(0)
	p.col[2] = things.OutfitColor(0)
	p.col[3] = things.OutfitColor(0)
	if oom := r.URL.Query().Get("oom"); oom != "" {
		oom2, _ := strconv.Atoi(oom)
		// ignore invalid oom

		if oom2 < int(things.OutfitOverlayMaskLast) {
			p.outfitOverlayMask = things.OutfitOverlayMask(oom2)
		}
	}
	for i := 0; i < 4; i++ {
		if col := r.URL.Query().Get(fmt.Sprintf("col%d", i)); col != "" {
			col2, _ := strconv.Atoi(col)
			// ignore invalid oom

			if col2 < 133 {
				p.col[i] = things.OutfitColor(col2)
			}
		}
	}

	generation := 1 // bump if the way we generate it changes
	mime := "image/png"
	etag := fmt.Sprintf(`W/"20211019:creature:%d:%08x:%08x:%d:%d:%d:%d.%d.%d.%d:%s"`, generation, th.SpriteSetSignature(), th.TibiaDatasetSignature(), idx, dir, fr, p.outfitOverlayMask, p.col[0], p.col[1], p.col[2], p.col[3], mime)
	if r.Header.Get("If-None-Match") == etag {
		w.Header().Set("Cache-Control", "public; max-age=36000") // 36000 = 10h
		w.Header().Set("ETag", etag)
		w.WriteHeader(http.StatusNotModified)
		return
	}

	img := cr.ColorizedCreatureFrame(fr, things.CreatureDirection(dir), p.outfitOverlayMask, p.col[:])
	if img == nil {
		http.Error(w, "bad image", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", mime)
	w.Header().Set("Cache-Control", "public; max-age=3600")
	w.Header().Set("ETag", etag)
	if s, err := os.Stat(tibiaSprPath); err == nil {
		// TODO: max of tibia.dat, tibia.spr, maybe more
		w.Header().Set("Last-Modified", s.ModTime().Format(http.TimeFormat))
	}
	w.WriteHeader(http.StatusOK)
	png.Encode(w, img)
}

func creatureGIFHandler(w http.ResponseWriter, r *http.Request) {
	creatureLock.Lock()
	defer creatureLock.Unlock()

	vars := mux.Vars(r)
	idx, err := strconv.Atoi(vars["idx"])
	if err != nil {
		http.Error(w, "idx not a number", http.StatusBadRequest)
		return
	}
	dir, err := strconv.Atoi(vars["dir"])
	if err != nil {
		http.Error(w, "dir not a number", http.StatusBadRequest)
		return
	}

	cr, err := th.CreatureWithClientID(uint16(idx), 854)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	g := gif.GIF{}

	// TODO: Can we do better? Can we calculate best palette for each frame?
	imgPalette := make([]color.Color, len(palette.WebSafe)+1)
	imgPalette[0] = image.Transparent
	copy(imgPalette[1:], palette.WebSafe)

	start := 1
	if cr.IdleAnim() {
		start = 0
	}

	var p struct {
		outfitOverlayMask things.OutfitOverlayMask
		col               [4]color.Color
	}
	p.col[0] = things.OutfitColor(0)
	p.col[1] = things.OutfitColor(0)
	p.col[2] = things.OutfitColor(0)
	p.col[3] = things.OutfitColor(0)
	if oom := r.URL.Query().Get("oom"); oom != "" {
		oom2, _ := strconv.Atoi(oom)
		// ignore invalid oom

		if oom2 < int(things.OutfitOverlayMaskLast) {
			p.outfitOverlayMask = things.OutfitOverlayMask(oom2)
		}
	}
	for i := 0; i < 4; i++ {
		if col := r.URL.Query().Get(fmt.Sprintf("col%d", i)); col != "" {
			col2, _ := strconv.Atoi(col)
			// ignore invalid oom

			if col2 < 133 {
				p.col[i] = things.OutfitColor(col2)
			}
		}
	}

	generation := 1 // bump if the way we generate it changes
	mime := "image/gif"
	etag := fmt.Sprintf(`W/"20211019:creature:%d:%08x:%08x:%d:%d:%d.%d.%d.%d:%s"`, generation, th.SpriteSetSignature(), th.TibiaDatasetSignature(), idx, dir, p.outfitOverlayMask, p.col[0], p.col[1], p.col[2], p.col[3], mime)
	if r.Header.Get("If-None-Match") == etag {
		w.Header().Set("Cache-Control", "public; max-age=36000") // 36000 = 10h
		w.Header().Set("ETag", etag)
		w.WriteHeader(http.StatusNotModified)
		return
	}

	for i := start; i < cr.AnimCount(); i++ {
		img := cr.ColorizedCreatureFrame(i, things.CreatureDirection(dir), p.outfitOverlayMask, p.col[:])
		if img == nil {
			http.Error(w, "bad image", http.StatusInternalServerError)
			return
		}

		pal := image.NewPaletted(img.Bounds(), imgPalette)

		draw.Draw(pal, img.Bounds(), img, image.ZP, draw.Over)
		g.Image = append(g.Image, pal)
		g.Delay = append(g.Delay, 50)
		g.Disposal = append(g.Disposal, gif.DisposalBackground)
		g.BackgroundIndex = 0 // image.Transparent
	}

	w.Header().Set("Content-Type", mime)
	w.Header().Set("Cache-Control", "public; max-age=3600")
	w.Header().Set("ETag", etag)
	if s, err := os.Stat(tibiaSprPath); err == nil {
		// TODO: max of tibia.dat, tibia.spr, maybe more
		w.Header().Set("Last-Modified", s.ModTime().Format(http.TimeFormat))
	}

	w.WriteHeader(http.StatusOK)
	gif.EncodeAll(w, &g)
}

func main() {
	setupFilePathFlags()
	flagutil.Parse()

	th = thingsOpen()
	r := mux.NewRouter()

	funcs := template.FuncMap{
		"N": iter.N,
		"N2": func(n1, n2 int) <-chan int {
			c := make(chan int)
			go func() {
				for i := n1; i < n2; i++ {
					c <- i
				}
				close(c)
			}()
			return c
		},
		"plusone": func(n int) int { return n + 1 },
		"add":     func(a, b int) int { return a + b },
		"mul":     func(a, b int) int { return a * b },
		"itemWithClientID": func(cid int) *things.Item {
			it, err := th.ItemWithClientID(uint16(cid), 854)
			if err != nil {
				panic(err)
			}
			return it
		},
		"item": func(sid int) *things.Item {
			it, err := th.Item(uint16(sid), 854)
			if err != nil {
				return nil
			}
			return it
		},
		"datasetColorToHex": func(col tdat.DatasetColor) string {
			r, g, b, _ := col.RGBA()
			return fmt.Sprintf("#%02x%02x%02x", uint8(r>>8), uint8(g>>8), uint8(b>>8))
		},
		"datasetColorIndexToHex": func(idx int) string {
			col := tdat.DatasetColor(idx)
			r, g, b, _ := col.RGBA()
			return fmt.Sprintf("#%02x%02x%02x", uint8(r>>8), uint8(g>>8), uint8(b>>8))
		},
		"outfitColorToHex": func(col things.OutfitColor) string {
			r, g, b, _ := col.RGBA()
			return fmt.Sprintf("#%02x%02x%02x", uint8(r>>8), uint8(g>>8), uint8(b>>8))
		},
		"outfitColorIndexToHex": func(idx int) string {
			col := things.OutfitColor(idx)
			r, g, b, _ := col.RGBA()
			return fmt.Sprintf("#%02x%02x%02x", uint8(r>>8), uint8(g>>8), uint8(b>>8))
		},
		"isNil": func(x interface{}) bool {
			if itm, ok := x.(*things.Item); ok {
				return itm == nil || !itm.ValidClientItem()
			}
			return x == nil
		},
	}
	itemTableTemplate := template.New("")
	itemTableTemplate = itemTableTemplate.Funcs(funcs)
	itemTableTemplate, err := itemTableTemplate.ParseFiles(paths.Find("itemtable.html"))

	if err != nil {
		glog.Errorf("not serving homepage, could not parse itemtable.html: %v", err)
	} else {
		r.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
			// REMOVE THIS begin
			itemTableTemplate := template.New("")
			itemTableTemplate = itemTableTemplate.Funcs(funcs)
			itemTableTemplate, err := itemTableTemplate.ParseFiles(paths.Find("itemtable.html"))
			if err != nil {
				glog.Errorf("not serving homepage, could not parse itemtable.html: %v", err)
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			// REMOVE THIS end

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

			params := struct {
				PG, PGMin, PGMax, PGSize int
			}{
				PG:     pg,
				PGMin:  pgMin,
				PGMax:  pgMax,
				PGSize: pgSize,
			}

			glog.Errorf("%v", itemTableTemplate.ExecuteTemplate(w, "itemtable.html", params))
			return
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
	}
	r.HandleFunc("/spr/{idx:[0-9]+}", sprHandler)
	r.HandleFunc("/item/{idx:[0-9]+}", itemHandler)
	r.HandleFunc("/item/c{idx:[0-9]+}", citemHandler)
	r.HandleFunc("/creature/{idx:[0-9]+}-{dir:[0-9]+}-{fr:[0-9]+}", creatureHandler)
	r.HandleFunc("/creature/{idx:[0-9]+}-{dir:[0-9]+}.gif", creatureGIFHandler)
	r.HandleFunc("/pic/{idx:[0-9]+}", picHandler)
	if htmlPath != "" {
		glog.Infof("serving %q as the /app/", htmlPath)
		r.HandleFunc("/app/Tibia.spr", func(w http.ResponseWriter, r *http.Request) {
			generation := 1 // bump if the way we generate it changes
			mime := "application/octet-stream"
			etag := fmt.Sprintf(`W/"spritefile:%d:%08x:%s"`, generation, th.SpriteSetSignature(), mime)
			if r.Header.Get("If-None-Match") == etag {
				w.Header().Set("Cache-Control", "public; max-age=36000") // 36000 = 10h
				w.Header().Set("ETag", etag)
				w.WriteHeader(http.StatusNotModified)
				return
			}

			w.Header().Set("Cache-Control", "public; max-age=36000") // 36000 = 10h
			w.Header().Set("ETag", etag)

			http.ServeFile(w, r, tibiaSprPath)
		})
		r.HandleFunc("/favicon.ico", func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Cache-Control", "public; max-age=36000") // 36000 = 10h
			http.ServeFile(w, r, htmlPath+"/favicon.ico")
		})
		r.HandleFunc("/sw.js", func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Cache-Control", "public; max-age=36000") // 36000 = 10h
			w.Header().Set("Content-Type", "application/javascript")
			http.ServeFile(w, r, htmlPath+"/sw.js")
		})
		r.PathPrefix("/app/").Handler(http.StripPrefix("/app/", http.FileServer(http.Dir(htmlPath))))

	}

	go func() {
		var m gameworld.MapDataSource
		if mapPath == ":test:" {
			m = gameworld.NewMapDataSource()
		} else {
			f, err := os.Open(mapPath)
			if err != nil {
				glog.Errorln("opening map file", err)
				return
			}
			m, err = otbm.New(f, th)
			if err != nil {
				glog.Errorln("reading map file", err)
				return
			}
			f.Close()
		}
		r.HandleFunc("/map", func(w http.ResponseWriter, r *http.Request) {
			t := th
			var tx, ty uint16
			var tbot, ttop uint8
			var tw, th int

			tx = 84
			ty = 84
			tbot = 7
			ttop = 0
			tw = 18
			th = 14

			if x := r.URL.Query().Get("x"); x != "" {
				txI, _ := strconv.Atoi(x)
				tx = uint16(txI)
			}
			if y := r.URL.Query().Get("y"); y != "" {
				tyI, _ := strconv.Atoi(y)
				ty = uint16(tyI)
			}
			if w := r.URL.Query().Get("w"); w != "" {
				tw, _ = strconv.Atoi(w)
			}
			if h := r.URL.Query().Get("h"); h != "" {
				th, _ = strconv.Atoi(h)
			}
			if bot := r.URL.Query().Get("bot"); bot != "" {
				tbotI, _ := strconv.Atoi(bot)
				tbot = uint8(tbotI)
			}
			if top := r.URL.Query().Get("top"); top != "" {
				ttopI, _ := strconv.Atoi(top)
				ttop = uint8(ttopI)
			}

			if tw > 70 {
				tw = 70
			}
			if th > 70 {
				th = 70
			}

			// TODO: more input validation! never allow for number inside CompositeMap to go negative, e.g.
			img := gameworld.CompositeMap(m, t, tx, ty, ttop, tbot, tw, th, 32, 32)
			if true {
				w.Header().Set("Content-Type", "image/png")
				w.WriteHeader(http.StatusOK)
				png.Encode(w, img)
			} else {
				w.Header().Set("Content-Type", "image/jpeg")
				w.WriteHeader(http.StatusOK)
				jpeg.Encode(w, img, &jpeg.Options{Quality: jpeg.DefaultQuality}) // jpeg.DefaultQuality})
			}

		})
	}()

	if *debugListenAddress != "" {
		// TODO(ivucica): have a mux that includes /debug URLs.
		go http.ListenAndServe(*debugListenAddress, nil)
	}

	glog.Infof("beginning to serve")
	glog.Fatal(http.ListenAndServe(*listenAddress, handlers.LoggingHandler(os.Stderr, r)))
}
