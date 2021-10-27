package web

import (
	"encoding/binary"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/gif"
	"image/jpeg"
	"image/png"
	"io"
	"net/http"
	"os"
	"strconv"
	"sync"

	"github.com/andybons/gogif"
	"github.com/golang/glog"
	"github.com/gorilla/mux"

	"badc0de.net/pkg/go-tibia/gameworld"
	"badc0de.net/pkg/go-tibia/spr"
	"badc0de.net/pkg/go-tibia/things"
)

type Handler struct {
	itemLock      sync.Mutex
	creatureLock  sync.Mutex
	th            *things.Things
	mapDataSource gameworld.MapDataSource

	tibiaSprPath string
	tibiaPicPath string
}

// NewHandler constructs web handler for the passed things. It also needs access
// to the .spr and .pic directly.
func NewHandler(th *things.Things, tibiaSprPath, tibiaPicPath string) *Handler {
	h := &Handler{
		th:           th,
		tibiaSprPath: tibiaSprPath,
		tibiaPicPath: tibiaPicPath,
	}
	return h
}

func (h *Handler) itemHandler(w http.ResponseWriter, r *http.Request) {
	h.itemLock.Lock()
	defer h.itemLock.Unlock()

	th := h.th

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
		http.Error(w, "image could not be generated", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", mime)
	w.Header().Set("Cache-Control", "public; max-age=36000") // 36000 = 10h
	w.Header().Set("ETag", etag)
	if s, err := os.Stat(h.tibiaSprPath); err == nil {
		// TODO: max of tibia.dat, tibia.spr, maybe more
		w.Header().Set("Last-Modified", s.ModTime().Format(http.TimeFormat))
	}
	w.WriteHeader(http.StatusOK)
	png.Encode(w, img)
}

func (h *Handler) citemHandler(w http.ResponseWriter, r *http.Request) {
	h.itemLock.Lock()
	defer h.itemLock.Unlock()

	th := h.th

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
	if s, err := os.Stat(h.tibiaSprPath); err == nil {
		// TODO: max of tibia.dat, tibia.spr, maybe more
		w.Header().Set("Last-Modified", s.ModTime().Format(http.TimeFormat))
	}
	w.WriteHeader(http.StatusOK)
	png.Encode(w, img)
}

func (h *Handler) creatureHandler(w http.ResponseWriter, r *http.Request) {
	h.creatureLock.Lock()
	defer h.creatureLock.Unlock()

	th := h.th

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
	if s, err := os.Stat(h.tibiaSprPath); err == nil {
		// TODO: max of tibia.dat, tibia.spr, maybe more
		w.Header().Set("Last-Modified", s.ModTime().Format(http.TimeFormat))
	}
	w.WriteHeader(http.StatusOK)
	png.Encode(w, img)
}

func (h *Handler) creatureGIFHandler(w http.ResponseWriter, r *http.Request) {
	h.creatureLock.Lock()
	defer h.creatureLock.Unlock()

	th := h.th

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

	quantizer := gogif.MedianCutQuantizer{NumColor: 255} // Up to 255 colors plus 1 space for transparency.
	for i := start; i < cr.AnimCount(); i++ {
		img := cr.ColorizedCreatureFrame(i, things.CreatureDirection(dir), p.outfitOverlayMask, p.col[:])
		if img == nil {
			http.Error(w, "bad image", http.StatusInternalServerError)
			return
		}

		pal := image.NewPaletted(img.Bounds(), nil)
		quantizer.Quantize(pal, img.Bounds(), img, image.ZP)

		// *sigh* gogif's MedianCutQuantizer doesn't provide for calculation of the palette
		// without also copying the image with dst.Set() on each pixel. it. Yes, we are thus
		// copying the image twice just to preserve transparency.
		//
		// Even worse, gogif's Quantize() is reading the whole image to calculate how many
		// pixels are in it before using dst.Set() on each pixel. Thanfully, our images are
		// tiny.

		// Create a version of paletted image with color.Transparent in it. That's the first
		// color so the empty image defaults to it.
		palTransparent := image.NewPaletted(img.Bounds(), append(color.Palette([]color.Color{color.Transparent}), pal.Palette...))

		// Now use draw.Draw() to create a version of the image which has the
		// MedianCutQuantizer's palette plus transparent color.
		draw.Draw(palTransparent, img.Bounds(), img, image.ZP, draw.Over)

		g.Image = append(g.Image, palTransparent)
		g.Delay = append(g.Delay, 50)
		g.Disposal = append(g.Disposal, gif.DisposalBackground)
		g.BackgroundIndex = 0 // image.Transparent
	}

	w.Header().Set("Content-Type", mime)
	w.Header().Set("Cache-Control", "public; max-age=3600")
	w.Header().Set("ETag", etag)
	if s, err := os.Stat(h.tibiaSprPath); err == nil {
		// TODO: max of tibia.dat, tibia.spr, maybe more
		w.Header().Set("Last-Modified", s.ModTime().Format(http.TimeFormat))
	}

	w.WriteHeader(http.StatusOK)
	gif.EncodeAll(w, &g)
}

func (h *Handler) sprHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	idx, err := strconv.Atoi(vars["idx"])
	if err != nil {
		http.Error(w, "idx not a number", http.StatusBadRequest)
		return
	}

	f, err := os.Open(h.tibiaSprPath)
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

func (h *Handler) picHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	idx, err := strconv.Atoi(vars["idx"])
	if err != nil {
		http.Error(w, "idx not a number", http.StatusBadRequest)
		return
	}

	f, err := os.Open(h.tibiaPicPath)
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
	var head spr.Header
	binary.Read(f, binary.LittleEndian, &head)

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
	etag := fmt.Sprintf(`W/"20211019:pic:%d:%08x:%d:%d.%d.%d.%d.%s"`, generation, head.Signature, idx, src.Min.X, src.Min.Y, src.Max.X, src.Max.Y, mime)
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
	if s, err := os.Stat(h.tibiaPicPath); err == nil {
		w.Header().Set("Last-Modified", s.ModTime().Format(http.TimeFormat))
	}
	w.WriteHeader(http.StatusOK)
	png.Encode(w, img)
}

func (h *Handler) mapHandler(w http.ResponseWriter, r *http.Request) {
	t := h.th
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
	img := gameworld.CompositeMap(h.mapDataSource, t, tx, ty, ttop, tbot, tw, th, 32, 32)
	if true {
		w.Header().Set("Content-Type", "image/png")
		w.WriteHeader(http.StatusOK)
		png.Encode(w, img)
	} else {
		w.Header().Set("Content-Type", "image/jpeg")
		w.WriteHeader(http.StatusOK)
		jpeg.Encode(w, img, &jpeg.Options{Quality: jpeg.DefaultQuality}) // jpeg.DefaultQuality})
	}

}

func (h *Handler) RegisterRoutes(r *mux.Router) {
	r.HandleFunc("/spr/{idx:[0-9]+}", h.sprHandler)
	r.HandleFunc("/item/{idx:[0-9]+}", h.itemHandler)
	r.HandleFunc("/item/c{idx:[0-9]+}", h.citemHandler)
	r.HandleFunc("/creature/{idx:[0-9]+}-{dir:[0-9]+}-{fr:[0-9]+}", h.creatureHandler)
	r.HandleFunc("/creature/{idx:[0-9]+}-{dir:[0-9]+}.gif", h.creatureGIFHandler)
	r.HandleFunc("/pic/{idx:[0-9]+}", h.picHandler)

}

func (h *Handler) RegisterMapRoute(r *mux.Router, mapDataSource gameworld.MapDataSource) {
	h.mapDataSource = mapDataSource
	r.HandleFunc("/map", h.mapHandler)
}
