package main

import (
	"flag"
	"fmt"
	"image"
	"image/color"
	"io"
	"os"
	"sync"

	"badc0de.net/pkg/flagutil/v1"

	tdat "badc0de.net/pkg/go-tibia/v1/dat"
	"badc0de.net/pkg/go-tibia/v1/imageprint"
	"badc0de.net/pkg/go-tibia/v1/otb/items"
	"badc0de.net/pkg/go-tibia/v1/spr"
	"badc0de.net/pkg/go-tibia/v1/things"

	"github.com/golang/glog"
	"github.com/nfnt/resize"
	"golang.org/x/crypto/ssh/terminal"
)

var (
	sprID   = flag.Int("spr", 0, "sprite to print")
	picID   = flag.Int("pic", 0, "pic to print")
	itemID  = flag.Int("item", 0, "server ID of item to print")
	citemID = flag.Int("citem", 0, "client ID of item to print")
	col256  = flag.Bool("col256", false, "whether to use 256 col instead of 24 bit")
	iterm   = flag.Bool("iterm", false, "whether to print with iterm escape code instead of 24 bit")
	blanks  = flag.Bool("blanks", true, "whether to just use colored blanks instead of some bad ascii art")
	col  = flag.Bool("color", true, "whether to use colorization escape sequences at all")

	creatureID = flag.Int("creature", 0, "ID of creature to print")

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

var (
	th *things.Things
)

func itemHandler(idx int) {
	itm, err := th.Item(uint16(idx), 854)
	if err != nil {
		return
	}

	img := itm.ItemFrame(0, 0, 0, 0)

	out(img)
}

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

func creatureHandler(idx int) {
	cr, err := th.CreatureWithClientID(uint16(idx), 854)
	if err != nil {
		fmt.Printf("err: %v", err)
		return
	}

	img := cr.ColorizedCreatureFrame(0, 2, 0, []color.Color{things.OutfitColor(130), things.OutfitColor(90), things.OutfitColor(25), things.OutfitColor(130)})

	out(img)
}

func out(img image.Image) {

	if w, h, err := terminal.GetSize(0); err == nil { // or int(os.Stdin.Fd())
		img = resize.Thumbnail(uint(w/2), uint(h), img, resize.Lanczos3)
	}

	if !*col {
		imageprint.PrintNoColor(img, *blanks)
	} else if *iterm {
		imageprint.PrintITerm(img, "image.png")
	} else if *col256 {
		imageprint.Print256Color(img, *blanks)
	} else {
		imageprint.Print24bit(img, *blanks)
	}
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

func main() {
	setupFilePathFlags()
	flagutil.Parse()
	flag.Set("logtostderr", "true")

	th = thingsOpen()

	if *sprID != 0 {
		sprHandler(*sprID)
	}
	if *itemID != 0 {
		itemHandler(*itemID)
	}
	if *citemID != 0 {
		citemHandler(*citemID, 0, 0, 0, 0)
	}
	if *creatureID != 0 {
		creatureHandler(*creatureID)
	}
	if *picID != 0 {
		picHandler(*picID)
	}
}
