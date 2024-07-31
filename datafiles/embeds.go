// +build go1.16

package datafiles

import "embed" // at least "import _ "embed"" is required

//go:embed itemtable.html
var itemsHTMLEmbed string

//go:embed outfittable.html
var outfitsHTMLEmbed string

//go:embed itemtable.html outfittable.html
var htmlTemplatesEmbed embed.FS
