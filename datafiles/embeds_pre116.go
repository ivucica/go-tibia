// +build !go1.16

package datafiles

import (
    "io/fs"
)

var itemsHTMLEmbed string
var outfitsHTMLEmbed string
var htmlTemplatesEmbed fs.FS
