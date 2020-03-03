# go-tibia: Tibia tools in Go

Just a toy project to see how far Go's built in crypto primitives can take me.

## gotserv: OTServ in Go

Main binary: `badc0de.net/pkg/go-tibia/cmd/gotserv`

So far implemented: stub login protocol, stub gameworld protocol which presents a map.

[Godoc documentation](https://godoc.org/badc0de.net/pkg/go-tibia/cmd/gotserv)

## Libraries

There's:

* [a spr reader](https://godoc.org/badc0de.net/pkg/go-tibia/v1/spr)
* [a dat reader](https://godoc.org/badc0de.net/pkg/go-tibia/v1/dat)
* [an otb reader](https://godoc.org/badc0de.net/pkg/go-tibia/v1/otb)
* [an items.otb reader](https://godoc.org/badc0de.net/pkg/go-tibia/v1/otb/items) built on top of the otb reader
* [a base network constructs library](https://godoc.org/badc0de.net/pkg/go-tibia/v1/net)
* [a login server](https://godoc.org/badc0de.net/pkg/go-tibia/v1/login)
* [a gameworld server](https://godoc.org/badc0de.net/pkg/go-tibia/v1/gameworld)
    * includes a toy, experimental compositor for a map into an `image.Image` incl lighting
* [an abstract representation of 'things'](https://godoc.org/badc0de.net/pkg/go-tibia/v1/things) such as items, creatures, etc, as an abstraction of items from items.otb, or .dat dataset, or otherwise
    * includes a toy, experimental compositor for items and creatures into `image.Image`

