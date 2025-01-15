// Binary wikiloader reads an XML-formatted dump of tibiawiki, obtainable from its `Special:Statistics` page.
//
// This is experimental reader working with 2024-02-15 19:23:22 version of the dump.
package main

import (
	"flag"
	"log"
	"os"

	"badc0de.net/pkg/go-tibia/xmls"
)

var (
	wikiDumpPath = flag.String("wiki_dump_path", "tibiawiki_pages_current.xml", "Path to the tibiawiki dump.")
)

func main() {
	flag.Parse()
	f, err := os.Open(*wikiDumpPath)
	if err != nil {
		panic(err)
	}

	e := xmls.NewWikiLoader(f)
	for {
		if e.Step() {
			log.Print("breaking")
			break
		}
	}
}
