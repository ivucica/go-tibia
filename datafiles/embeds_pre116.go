// +build !go1.16

package datafiles

import (
    // "io/fs" // also introduced in 1.16
)

var itemsHTMLEmbed string
var outfitsHTMLEmbed string
var htmlTemplatesEmbed interface{} //fs.FS
