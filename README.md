# go-tibia: Tibia tools in Go

Just a toy project to see how far Go's built in crypto primitives can take me,
with some experimentation on how decent the Go interface abstractions, and how
useful Go's WASM support is.

Copyright © 2017-2024 Ivan Vucica. See the [LICENSE](LICENSE) for licensing
information.

## gotserv: OTServ in Go

Main binary: `badc0de.net/pkg/go-tibia/cmd/gotserv`

So far implemented: stub login protocol, stub gameworld protocol which presents
a map, some moving code. Other players can be seen, but updates from them won't
propagate yet.

A debug webserver can be enabled, which reuses some of the gotweb code to paint
a live representation of a portion of the map upon request.

[Godoc documentation](https://godoc.org/badc0de.net/pkg/go-tibia/cmd/gotserv)

## gotweb

Main binary: `badc0de.net/pkg/go-tibia/cmd/gotweb`

Displays an index of items and creatures. Serves pic file's and spr file's
individual components. Serves the requested sub-portion of the loaded map,
composited with lighting.

Serves gotwebfe via a service worker, ensuring service worker is served with
up-to-date cache keys so new files get fetched as needed.

[Godoc documentation](https://godoc.org/badc0de.net/pkg/go-tibia/cmd/gotweb)

## gotwebfe

Main binary: `badc0de.net/pkg/go-tibia/cmd/gotwebfe`

Code intended to be compiled into WASM and run in-browser. Loads spr, pic,
dat, items.xml and otbm files, and can render the map using the compositor.

Portion of the implementation is in HTML and CSS files in `datafiles/html`.
It's currently a PWA: it'll cache spr, pic, dat etc offline.

[Godoc documentation](https://godoc.org/badc0de.net/pkg/go-tibia/cmd/gotwebfe)

## itemprint

Main binary: `badc0de.net/pkg/go-tibia/cmd/itemprint`

Draws pic, spr and even items as composited using spr+dat files into terminal.
It can use various methods: kitty's image drawing, iTerm2's png image drawing,
true-color colored characters, 256-color colored characters, dumb 'intensity'
ascii 'art'. The images get shrunk as needed.

[Godoc documentation](https://godoc.org/badc0de.net/pkg/go-tibia/cmd/itemprint)

## Libraries

There's:

* [a spr reader](https://godoc.org/badc0de.net/pkg/go-tibia/spr)
* [a dat reader](https://godoc.org/badc0de.net/pkg/go-tibia/dat)
* [an otb reader](https://godoc.org/badc0de.net/pkg/go-tibia/otb)
* [an items.otb reader](https://godoc.org/badc0de.net/pkg/go-tibia/otb/items) built on top of the otb reader
* [an .otbm reader](https://godoc.org/badc0de.net/pkg/go-tibia/otb/map) built on top of the otb reader
* [a base network constructs library](https://godoc.org/badc0de.net/pkg/go-tibia/net)
* [a login server](https://godoc.org/badc0de.net/pkg/go-tibia/login)
* [a gameworld server](https://godoc.org/badc0de.net/pkg/go-tibia/gameworld)
* [a compositor for the map](https://godoc.org/badc0de.net/pkg/go-tibia/compositor), a toy compositor painting a map into an `image.Image`, including lighting
    * [a browser DOM compositor for the map](https://godoc.org/badc0de.net/pkg/go-tibia/compositor/dom), a toy compositor assembling a map out of DOM objects using `syscall/js` (i.e. for WASM environment); the actual approach (a single `<img>`, or many `<img>` representing tiles, or many `<img>`s representing items on tiles) is an implementation
* [an abstract representation of 'things'](https://godoc.org/badc0de.net/pkg/go-tibia/things) such as items, creatures, etc, as an abstraction of items from items.otb, or .dat dataset, or otherwise
    * includes a toy, experimental compositor for items and creatures into `image.Image`
    * also can load "all" files from "default paths"
* [an .xml loader](https://godoc.org/badc0de.net/pkg/go-tibia/xmls), currently loading only OpenTibia's outfits.xml
* [a secret constants package](https://godoc.org/badc0de.net/pkg/go-tibia/secrets), currently containing only OpenTibia encryption "secret" constants
* [a web handlers collection](https://godoc.org/badc0de.net/pkg/go-tibia/web), used to share some URL path handlers between `gotwebfe` and `gotserv`

Some helpers:

* [path handler](https://godoc.org/badc0de.net/pkg/go-tibia/paths) dealing with finding the files in the filesystem and setting up flags as needed
* [image printer](https://godoc.org/badc0de.net/pkg/go-tibia/imageprint) used in `itemprint` binary and in some tests

