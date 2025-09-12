package main

import (
	"bytes"
	"html/template"
	_ "net/http/pprof" // Default mux should not be served publicly; it's actually hidden behind a flag.

	"crypto/md5"
	"encoding/binary"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"

	"badc0de.net/pkg/flagutil"

	tdat "badc0de.net/pkg/go-tibia/dat"
	"badc0de.net/pkg/go-tibia/gameworld"    // for map compositor
	otbm "badc0de.net/pkg/go-tibia/otb/map" // for map loader
	"badc0de.net/pkg/go-tibia/paths"
	"badc0de.net/pkg/go-tibia/spr"
	"badc0de.net/pkg/go-tibia/things"
	"badc0de.net/pkg/go-tibia/things/full"
	"badc0de.net/pkg/go-tibia/web"
	"badc0de.net/pkg/go-tibia/xmls"

	"github.com/bradfitz/iter"
	"github.com/golang/glog"
	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
)

var (
	listenAddress      = flag.String("listen_address", ":8080", "http listen address for gotweb")
	debugListenAddress = flag.String("debug_listen_address", "", "http listen address for pprof (and other stuff registered on default serve mux)")

	vapidPrivate           = flag.String("vapid_private", "", "vapid private key for use with push notifications; empty disables push to service worker; use cmd/vapidgen to generate")
	vapidPublic            = flag.String("vapid_public", "", "vapid public key for use with push notifications; empty disables push to service worker; use cmd/vapidgen to generate")
	vapidSubscriptionsPath string

	tibiaPicPath string
	mapPath      string

	htmlPath        string
	appHTMLPath     string
	itemsHTMLPath   string
	outfitsHTMLPath string
)

type ReadSeekerCloser = io.ReadSeekCloser

func picOpen() (ReadSeekerCloser, error) {
	return os.Open(tibiaPicPath)
}

func setupFilePathFlags() {
	full.SetupFilePathFlags()
	paths.SetupFilePathFlag("Tibia.pic", "tibia_pic_path", &tibiaPicPath)
	paths.SetupFilePathFlag(":test:", "map_path", &mapPath)
	paths.SetupFilePathFlag("itemtable.html", "items_index_html_path", &itemsHTMLPath)
	paths.SetupFilePathFlag("outfittable.html", "outfits_index_html_path", &outfitsHTMLPath)
	paths.SetupFilePathFlag("html/index.html", "app_html_path", &appHTMLPath)
	paths.SetupFilePathFlag("vapid_subscriptions.json", "writable_vapid_subscriptions_json_path", &vapidSubscriptionsPath)
	htmlPath = filepath.Dir(appHTMLPath)
}

func thingsOpen() *things.Things {
	th, _ := full.FromDefaultPaths(true)
	return th
}

var (
	th         *things.Things
	outfitsXML *xmls.Outfits
)

func main() {
	setupFilePathFlags()
	flagutil.Parse()

	if mapPath == "" {
		mapPath = ":test:"
	}

	th = thingsOpen()
	r := mux.NewRouter()

	f, err := paths.Open("outfits.xml") // Paths will use the outfits.xml flag.
	if err != nil {
		glog.Errorf("could not open outfits xml: %v", err)
		outfitsXML = &xmls.Outfits{}
	} else {
		outfits, err := xmls.ReadOutfits(f)
		if err != nil {
			glog.Errorf("could not parse outfits xml: %v", err)
			outfitsXML = &xmls.Outfits{}
		} else {
			outfitsXML = &outfits
		}
	}
	f.Close()

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
		"itemWithClientIDX": func(clsrvIDX int) *things.Item {
			// Slightly strange but this is essentially:
			// - we want to paint all OTB items that have known client ID, but skip the others
			// - we have an array of all OTB items with known client IDs mapping to all the OTB items
			// - so we have a function that offsets inside the "known client IDs", then finds the relevant OTB item instead
			it, err := th.ItemWithSequentialClientID(uint16(clsrvIDX), 854)
			if err != nil {
				panic(err)
			}
			return it
		},
		"itemWithServerIDX": func(srvIDX int) *things.Item {
			// same as *ClientIDX except this is only useful if we would know some OTB items don't have server ID set
			// (is that even valid?)
			it, err := th.ItemWithSequentialServerID(uint16(srvIDX), 854)
			if err != nil {
				panic(err)
			}
			return it
		},
		"itemWithOTBIDX": func(otbIDX int) *things.Item {
			// same as *ClientIDX except this is only useful if we would want to render even those OTB items without server ID set
			// (is lacking server ID even valid?)
			it, err := th.ItemWithSequentialOTBIDX(otbIDX, 854)
			if err != nil {
				panic(err)
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
	itemTableTemplate, err = itemTableTemplate.ParseFiles(itemsHTMLPath)

	if err != nil {
		glog.Errorf("not serving homepage, could not parse itemtable.html: %v", err)
	} else {
		r.HandleFunc("/items/sitemap.xml", func(w http.ResponseWriter, r *http.Request) {
			// items sitemap
			// we also serve https://developers.google.com/search/docs/crawling-indexing/sitemaps/image-sitemaps
			// needs to come before sid

			// we can add up to 50k items, which is less than otb and dat have
			us := &SitemapURLSet{}
			for i := 0; i < th.ClientItemCount(854); i++ {
				itm, err := th.ItemWithSequentialClientID(uint16(i), 854)
				if err != nil {
					continue
				}
				us.URL = append(us.URL,
					SitemapURL{
						Loc: fmt.Sprintf("https://go-tibia.badc0de.net/items/%d", itm.ServerID()), // DO NOT SUBMIT
						Image: []SitemapURLImage{
							{
								Loc: fmt.Sprintf("https://go-tibia.badc0de.net/item/%d", itm.ServerID()), // DO NOT SUBMIT
							},
						},
					},
				)
			}

			us.Write(w, r)
		})

		fgen := func(pgSize int, defaultPg int, clientVersion uint16, serverIDX, knownClientIDArrayIDX bool) func(w http.ResponseWriter, r *http.Request) {
			f := func(w http.ResponseWriter, r *http.Request) {
				// REMOVE THIS begin
				// removeable because used only to reload itemtable.html during dev
				itemTableTemplate := template.New("")
				itemTableTemplate = itemTableTemplate.Funcs(funcs)
				itemTableTemplate, err := itemTableTemplate.ParseFiles(itemsHTMLPath)
				if err != nil {
					glog.Errorf("not serving homepage, could not parse itemtable.html: %v", err)
					http.Error(w, err.Error(), http.StatusInternalServerError)
					return
				}
				// REMOVE THIS end

				pg := defaultPg
				// pgSize := 25

				if pgStr := r.URL.Query().Get("page"); pgStr != "" && defaultPg == 0 {
					if pgConv, err := strconv.Atoi(pgStr); err == nil {
						pg = pgConv - 1
					}
				}

				w.Header().Set("Content-Type", "text/html")

				var itemMin, itemMax int // item = list item; either client item IDs or server item IDs
				var itemCount int        // item count across all pages
				if serverIDX {
					// rendering server items.
					//
					// server IDs have skips in them, so we use our own indexing offset
					itemMin = 0
					itemMax = th.ItemCount(clientVersion)
					itemCount = th.ItemCount(clientVersion)
				} else if knownClientIDArrayIDX {
					// rendering server items.
					//
					// server IDs have skips in them, so we use our own indexing offset
					// and in fact, even though we are rendering server and not client items, we will
					// only render those known under current clientVersion
					itemMin = 0
					itemMax = th.ClientItemCount(clientVersion)
					itemCount = th.ClientItemCount(clientVersion)
				} else {
					switch clientVersion {
					case 854:
						// client IDs for 8.54
						itemMin = int(th.MinItemClientID(clientVersion))
						itemMax = int(th.MaxItemClientID(clientVersion))
						itemCount = int(th.Temp__DATItemCount(clientVersion))
					default:
						http.NotFound(w, r)
					}
				}

				pgMin := 0
				pgMax := (itemMax - itemMin) / pgSize

				if pg < pgMin {
					pg = pgMin
				}
				if pg > pgMax {
					pg = pgMax
				}

				params := struct {
					PG, PGMin, PGMax, PGSize int
					Client                   uint16
					ServerIDX                bool // request render of server IDX version of the page
					KnownClientIDX           bool // request render of server IDX version of the page but based only on known client IDs
					ItemCount                int  // total count of items across pages
				}{
					PG:     pg,
					PGMin:  pgMin,
					PGMax:  pgMax,
					PGSize: pgSize,

					ItemCount:      itemCount,
					Client:         clientVersion,
					ServerIDX:      serverIDX,
					KnownClientIDX: knownClientIDArrayIDX,
				}

				err = itemTableTemplate.ExecuteTemplate(w, filepath.Base(itemsHTMLPath), params)
				if err != nil {
					glog.Errorf("failed to execute itemtable.html: %v", err)
				}
			}
			return f
		}
		r.HandleFunc("/", fgen(25, 0, 854, false, true))
		r.HandleFunc("/items/", fgen(25, 0, 854, false, true))
		r.HandleFunc("/items/{sid:[0-9]+}", func(w http.ResponseWriter, r *http.Request) {
			// Ugly hack: until we have an item details page, we display just a single
			// row in the tableview.
			//
			// But because our table displays with otb array index, we first have to
			// map serverid to otb array idx.
			sid, err := strconv.Atoi(mux.Vars(r)["sid"])

			if err != nil {
				glog.Errorf("item id %q invalid: %v", mux.Vars(r)["sid"], err)
				http.NotFound(w, r)
				return
			}

			//sidx := th.Temp__GetServerItemArrayOffsetInOTB(uint16(sid), 854)
			//if sidx < 0 || sidx > th.ItemCount(854) {
			kciaidx := th.Temp__GetKnownClientIDItemArrayOffsetInOTB(uint16(sid), 854)
			if kciaidx < 0 || kciaidx > th.ClientItemCount(854) {
				glog.Errorf("kciaidx %d invalid for sid %d", kciaidx, sid)
				http.NotFound(w, r)
				return
			}

			// fgen(1, sid-int(th.MinItemServerID(854)), 854, true, false)(w, r)
			// fgen(1, sidx, 854, true, false)(w, r)
			fgen(1, kciaidx, 854, false, true)(w, r)

		})

		r.HandleFunc("/citems/854/", fgen(25, 0, 854, false, false))
		r.HandleFunc("/citems/854/item/", fgen(25, 0, 854, false, false))
		r.HandleFunc("/citems/854/item/{cid:[0-9]+}", func(w http.ResponseWriter, r *http.Request) {
			// Ugly hack: until we have an item details page, we display just a single
			// row in the tableview.
			cid, err := strconv.Atoi(mux.Vars(r)["cid"])
			cid16 := uint16(cid)
			if err != nil || cid16 < th.MinItemClientID(854) || cid16 > th.MaxItemClientID(854) {
				http.NotFound(w, r)
			} else {
				fgen(1, cid-int(th.MinItemClientID(854)), 854, false, false)(w, r)
			}
		})
	}

	outfitTableTemplate := template.New("")
	outfitTableTemplate = outfitTableTemplate.Funcs(funcs)
	outfitTableTemplate, err = outfitTableTemplate.ParseFiles(outfitsHTMLPath)

	if err != nil {
		glog.Errorf("not serving /outfits/, could not parse outfittable.html: %v", err)
	} else {
		r.HandleFunc("/outfits/", func(w http.ResponseWriter, r *http.Request) {
			// REMOVE THIS begin
			// removeable because used only to reload outfittable.html during dev
			outfitTableTemplate := template.New("")
			outfitTableTemplate = outfitTableTemplate.Funcs(funcs)
			outfitTableTemplate, err := outfitTableTemplate.ParseFiles(outfitsHTMLPath)
			if err != nil {
				glog.Errorf("not serving homepage, could not parse outfittable.html: %v", err)
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			// REMOVE THIS end

			w.Header().Set("Content-Type", "text/html")

			params := struct {
				OutfitsXML *xmls.Outfits
			}{
				OutfitsXML: outfitsXML,
			}

			err = outfitTableTemplate.ExecuteTemplate(w, filepath.Base(outfitsHTMLPath), params)
			if err != nil {
				glog.Errorf("failed to execute outfittable.html: %v", err)
			}
		})
	}

	r.HandleFunc("/sitemap.xml", func(w http.ResponseWriter, r *http.Request) {
		// sitemap index

		si := &SitemapIndex{
			Sitemap: []SitemapIndexSitemap{
				{
					Loc: "https://go-tibia.badc0de.net/items/sitemap.xml", // DO NOT SUBMIT
				},
				{
					Loc: "https://go-tibia.badc0de.net/sitemap-root.xml",
				},
			},
		}
		si.Write(w, r)
	})

	r.HandleFunc("/sitemap-root.xml", func(w http.ResponseWriter, r *http.Request) {
		// base sitemap

		us := &SitemapURLSet{
			URL: []SitemapURL{
				{
					Loc: "https://go-tibia.badc0de.net/app/",
				},
				{
					Loc: "https://go-tibia.badc0de.net/items/",
				},
				{
					Loc: "https://go-tibia.badc0de.net/outfits/",
				},
				{
					Loc: "https://go-tibia.badc0de.net/citems/854/",
				},
				{
					Loc: "https://go-tibia.badc0de.net/citems/854/item/",
				},
			},
		}

		us.Write(w, r)
	})

	r.HandleFunc("/robots.txt", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "Sitemap: /sitemap.xml\n")
		// fmt.Fprintf(w, "User-Agent: *\nDisallow: /\n")
	})

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
		r.HandleFunc("/app/Tibia.dat", func(w http.ResponseWriter, r *http.Request) {
			generation := 1 // bump if the way we generate it changes
			mime := "application/octet-stream"
			etag := fmt.Sprintf(`W/"datafile:%d:%08x:%s"`, generation, th.TibiaDatasetSignature(), mime)
			if r.Header.Get("If-None-Match") == etag {
				w.Header().Set("Cache-Control", "public; max-age=36000") // 36000 = 10h
				w.Header().Set("ETag", etag)
				w.WriteHeader(http.StatusNotModified)
				return
			}

			w.Header().Set("Cache-Control", "public; max-age=36000") // 36000 = 10h
			w.Header().Set("ETag", etag)

			http.ServeFile(w, r, full.PathFlagValue(full.FlagTibiaDatPath))
		})
		r.HandleFunc("/app/Tibia.pic", func(w http.ResponseWriter, r *http.Request) {
			generation := 1 // bump if the way we generate it changes
			mime := "application/octet-stream"

			f, err := paths.Open("Tibia.pic")
			if err != nil {
				http.Error(w, "failed to open pic file", http.StatusNotFound)
				return
			}
			defer f.Close()

			f.Seek(0, io.SeekStart)
			var head spr.Header
			binary.Read(f, binary.LittleEndian, &head)

			etag := fmt.Sprintf(`W/"picfile:%d:%08x:%s"`, generation, head.Signature, mime)
			if r.Header.Get("If-None-Match") == etag {
				w.Header().Set("Cache-Control", "public; max-age=36000") // 36000 = 10h
				w.Header().Set("ETag", etag)
				w.WriteHeader(http.StatusNotModified)
				return
			}

			w.Header().Set("Cache-Control", "public; max-age=36000") // 36000 = 10h
			w.Header().Set("ETag", etag)

			http.ServeFile(w, r, paths.Find("Tibia.pic"))
		})
		if mapPath != "" {
			r.HandleFunc("/app/map.otbm", func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Cache-Control", "public; max-age=36000") // 36000 = 10h
				http.ServeFile(w, r, mapPath)
			})
			glog.Warningf("map %q is served unauthenticated /app/map.otbm, careful if this is not intended", mapPath)
		}
		r.HandleFunc("/app/items.otb", func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Cache-Control", "public; max-age=36000") // 36000 = 10h
			http.ServeFile(w, r, paths.Find("items.otb"))
		})
		r.HandleFunc("/app/items.xml", func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Cache-Control", "public; max-age=36000") // 36000 = 10h
			http.ServeFile(w, r, paths.Find("items.xml"))
		})
		r.HandleFunc("/app/outfits.xml", func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Cache-Control", "public; max-age=36000") // 36000 = 10h
			http.ServeFile(w, r, paths.Find("outfits.xml"))
		})
		r.HandleFunc("/app/main.wasm", func(w http.ResponseWriter, r *http.Request) {

			etag := genericFileEtag("html/main.wasm", "wasmfile", w, r)
			if etag == "" {
				// http error/headers already set in genericFileEtag
				return
			}
			w.Header().Set("Cache-Control", "public; max-age=30") // 30 = 0.5 min
			w.Header().Set("ETag", etag)

			http.ServeFile(w, r, paths.Find("html/main.wasm"))
		})
		r.HandleFunc("/favicon.ico", func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Cache-Control", "public; max-age=36000") // 36000 = 10h
			http.ServeFile(w, r, htmlPath+"/favicon.ico")
		})
		r.HandleFunc("/app/_share-target-handler", func(w http.ResponseWriter, r *http.Request) {
			if r.Method != "POST" {
				http.Error(w, "only POST allowed", http.StatusMethodNotAllowed)
				return
			}
			// Not supposed to actually hit the backend since sw.js is supposed
			// to intercept it. And the endpoint is meant to be hit only when
			// PWA has something to receive as a target, i.e. it's been
			// installed.
			//
			// Still, let's return 204 if we do get hit. For now.
			//
			// Add cache control none.
			w.Header().Set("Cache-Control", "no-store")
			w.Header().Set("Expires", "0")
			w.WriteHeader(http.StatusNoContent)
		})
		r.HandleFunc("/sw.js", func(w http.ResponseWriter, r *http.Request) {
			f, err := os.Open(htmlPath + "/sw.js")
			if err != nil {
				http.Error(w, "404", http.StatusNotFound)
				return
			}

			buf := &bytes.Buffer{}
			_, err = io.Copy(buf, f)
			fi, err := f.Stat()
			if err != nil {
				http.Error(w, "500", http.StatusInternalServerError)
				return
			}
			f.Close()

			fiWasm, err := os.Stat(paths.Find("html/main.wasm")) // TODO: this is likely in htmlPath + "/main.wasm"
			if err != nil {
				http.Error(w, "500", http.StatusInternalServerError)
				return
			}

			fiIndexHTML, err := os.Stat(paths.Find("html/index.html")) // TODO: use appHTMLPath
			if err != nil {
				http.Error(w, "500", http.StatusInternalServerError)
				return
			}

			b := bytes.Replace(buf.Bytes(), []byte("%GO-TIBIA-CACHE-STORAGE-KEY%"), []byte(fmt.Sprintf("gotwebfe-cache-%d-%d-%d", fi.ModTime().UnixNano(), fiWasm.ModTime().UnixNano(), fiIndexHTML.ModTime().UnixNano())), -1)
			b = bytes.Replace(b, []byte("%GO-TIBIA-DATA-CACHE-STORAGE-KEY%"), []byte(fmt.Sprintf("gotwebfe-cache-data-%x-%x-omittinghashforpic", th.TibiaDatasetSignature(), th.SpriteSetSignature())), -1)
			replaced := bytes.NewReader(b)

			etag := genericFileEtagForContent(htmlPath+"/sw.js", "javascript", replaced, w, r)
			if etag == "" {
				// http error/headers already set in genericFileEtag
				return
			}

			replaced.Seek(0, io.SeekStart) // ignoring return values, should never fail for bytes.Reader

			w.Header().Set("Cache-Control", "public; max-age=30") // 30 = 0.5 min
			w.Header().Set("ETag", etag)
			w.Header().Set("Content-Type", "application/javascript")
			http.ServeContent(w, r, "sw.js", fi.ModTime(), replaced)
		})
		r.PathPrefix("/app/").Handler(http.StripPrefix("/app/", http.FileServer(http.Dir(htmlPath))))

	}

	h := web.NewHandler(th, full.PathFlagValue(full.FlagTibiaSprPath), tibiaPicPath)
	h.RegisterRoutes(r)

	go func() {
		var m gameworld.MapDataSource
		if mapPath == ":test:" {
			m = gameworld.NewMapDataSource()
		} else if mapPath == "" {
			glog.Warningf("mappath passed is empty, despite default being :test:; assuming :test:")
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

	if *vapidPrivate != "" {
		glog.Errorf("vapid private key specified, but push notifications are not implemented yet")
	}
	if *vapidPublic != "" {
		glog.Errorf("vapid public key specified, but push notifications are not implemented yet")
	}
	if *vapidPrivate != "" && *vapidPublic != "" {
		// not functional yet.
		f, err := os.Open(vapidSubscriptionsPath)
		var sm *web.SubscriptionManager
		if err != nil {
			sm, err = web.NewSubscriptionManager(nil)
			if err != nil {
				sm = nil
				glog.Errorf("web push notification subscription manager could not be spun up: %v", err)
			}
		} else {
			sm, err = web.NewSubscriptionManager(f)
			if err != nil {
				sm = nil
				glog.Errorf("web push notification subscription manager could not be spun up: %v", err)
			}
			f.Close()
		}

		if sm != nil {
			h.RegisterSubscriptionCreateRoute(r, sm)
		}
	}

	if *debugListenAddress != "" {
		// TODO(ivucica): have a mux that includes /debug URLs.
		go http.ListenAndServe(*debugListenAddress, nil)
	}

	glog.Infof("beginning to serve")
	glog.Fatal(http.ListenAndServe(*listenAddress, handlers.LoggingHandler(os.Stderr, r)))
}

func genericFileEtagForContent(fn, kind string, content io.ReadSeeker, w http.ResponseWriter, r *http.Request) string {
	// TODO: do not set http error here
	generation := 1
	hash := md5.New()
	_, err := io.Copy(hash, content)
	if err != nil {
		http.Error(w, "500", http.StatusInternalServerError)
		return ""
	}

	etag := fmt.Sprintf(`W/"%s:%d:%x"`, kind, generation, hash.Sum(nil))
	if r.Header.Get("If-None-Match") == etag {
		w.Header().Set("Cache-Control", "public; max-age=30") // 30 = 0.5 min
		w.Header().Set("ETag", etag)
		w.WriteHeader(http.StatusNotModified)
		return ""
	}
	return etag
}

func genericFileEtag(fn, kind string, w http.ResponseWriter, r *http.Request) string {
	// TODO: do not set http error here
	f, err := paths.Open(fn)
	if err != nil {
		http.Error(w, "404", http.StatusNotFound)
		return ""
	}
	defer f.Close()

	return genericFileEtagForContent(fn, kind, f, w, r)
}
