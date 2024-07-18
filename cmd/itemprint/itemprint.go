package main

import (
	"flag"
	"io"

	"badc0de.net/pkg/flagutil"
	"github.com/common-nighthawk/go-figure"

	"badc0de.net/pkg/go-tibia/paths"
)

var (
	sprID    = flag.Int("spr", 0, "sprite to print")
	picID    = flag.Int("pic", 0, "pic to print")
	itemID   = flag.Int("item", 0, "server ID of item to print")
	citemID  = flag.Int("citem", 0, "client ID of item to print")
	col256   = flag.Bool("col256", false, "whether to use 256 col instead of 24 bit")
	iterm    = flag.Bool("iterm", false, "whether to print with iterm escape code instead of 24 bit")
	rasterm  = flag.Bool("rasterm", true, "whether to print using the rasterm library")
	blanks   = flag.Bool("blanks", true, "whether to just use colored blanks instead of some bad ascii art")
	col      = flag.Bool("color", true, "whether to use colorization escape sequences at all")
	downsize = flag.Bool("downsize", true, "whether to downsize to terminal size")

	creatureID = flag.Int("creature", 0, "ID of creature to print")

	mapX   = flag.Int("map_x", 84, "x offset on the map (left)")
	mapY   = flag.Int("map_y", 84, "y offset on the map (top)")
	mapW   = flag.Int("map_w", 18, "width of rendered map")
	mapH   = flag.Int("map_h", 14, "height of rendered map")
	mapBot = flag.Int("map_bot", 7, "bottom of the rendered map")
	mapTop = flag.Int("map_top", 0, "top of the rendered map")

	itemsOTBPath string
	itemsXMLPath string
	tibiaDatPath string
	tibiaSprPath string
	tibiaPicPath string
	mapPath      = flag.String("map_path", "", "path to an otbm file (e.g. 'datafiles/map.otb' or ':test:'); if empty map is not loaded or rendered")
)

type ReadSeekerCloser = io.ReadSeekCloser

func setupFilePathFlags() {
	paths.SetupFilePathFlag("items.otb", "items_otb_path", &itemsOTBPath)
	paths.SetupFilePathFlag("items.xml", "items_xml_path", &itemsXMLPath)
	paths.SetupFilePathFlag("Tibia.dat", "tibia_dat_path", &tibiaDatPath)
	paths.SetupFilePathFlag("Tibia.spr", "tibia_spr_path", &tibiaSprPath)
	paths.SetupFilePathFlag("Tibia.pic", "tibia_pic_path", &tibiaPicPath)
}

func main() {
	setupFilePathFlags()
	flagutil.Parse()
	flag.Set("logtostderr", "true")

	th = thingsOpen()

	fig := figure.NewColorFigure("go-tibia itemprint", "", "green", true)
	fig.Print()

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
	if *mapPath != "" {
		mapHandler(*mapPath, *mapX, *mapY, *mapW, *mapH, *mapBot, *mapTop)
	}
}
