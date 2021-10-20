package main

import (
	"html/template"
	_ "net/http/pprof" // Default mux should not be served publicly; it's actually hidden behind a flag.

	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"

	"badc0de.net/pkg/flagutil"

	tdat "badc0de.net/pkg/go-tibia/dat"
	"badc0de.net/pkg/go-tibia/gameworld" // for map compositor
	"badc0de.net/pkg/go-tibia/otb/map" // for map loader
	"badc0de.net/pkg/go-tibia/paths"
	"badc0de.net/pkg/go-tibia/things"
	"badc0de.net/pkg/go-tibia/things/full"
	"badc0de.net/pkg/go-tibia/web"

	"github.com/bradfitz/iter"
	"github.com/golang/glog"
	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
)

var (
	listenAddress      = flag.String("listen_address", ":8080", "http listen address for gotweb")
	debugListenAddress = flag.String("debug_listen_address", "", "http listen address for pprof (and other stuff registered on default serve mux)")

	tibiaPicPath string
	mapPath      string
	htmlPath     string
)

type ReadSeekerCloser interface {
	io.Reader
	io.Seeker
	io.Closer
}

func picOpen() (ReadSeekerCloser, error) {
	return os.Open(tibiaPicPath)
}

func setupFilePathFlags() {
	full.SetupFilePathFlags()
	paths.SetupFilePathFlag("Tibia.pic", "tibia_pic_path", &tibiaPicPath)
	paths.SetupFilePathFlag("map.otbm", "map_path", &mapPath)
	paths.SetupFilePathFlag("html/index.html", "index_html_path", &htmlPath)
	htmlPath = filepath.Dir(htmlPath)
}

func thingsOpen() *things.Things {
	th, _ := full.FromDefaultPaths(true)
	return th
}

var (
	th *things.Things
)

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

			http.ServeFile(w, r, full.PathFlagValue(full.FlagTibiaSprPath))
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

	h := web.NewHandler(th, full.PathFlagValue(full.FlagTibiaSprPath), tibiaPicPath)
	h.RegisterRoutes(r)

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
		h.RegisterMapRoute(r, m)
	}()

	if *debugListenAddress != "" {
		// TODO(ivucica): have a mux that includes /debug URLs.
		go http.ListenAndServe(*debugListenAddress, nil)
	}

	glog.Infof("beginning to serve")
	glog.Fatal(http.ListenAndServe(*listenAddress, handlers.LoggingHandler(os.Stderr, r)))
}
