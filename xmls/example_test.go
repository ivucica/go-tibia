package xmls

import (
	"bytes"
	"fmt"
)

// Example demonstrates how to load outfits from the outfits.xml file.
func Example() {
	o := bytes.NewReader([]byte(`<?xml version="1.0"?>
<outfits>
	<outfit id="1" premium="0">
		<list type="female" looktype="136" name="Citizen"/>
		<list type="male" looktype="128" name="Citizen"/>
	</outfit>
</outfits>`))
	outfits, err := ReadOutfits(o)
	if err != nil {
		panic(err)
	}

	fmt.Println(outfits.Outfit[0].List[0].Name)
	// Output:
	// Citizen
}

// readerReadCloser wraps a bytes.Reader and adds a no-op Close
type readerReadCloser struct {
	*bytes.Reader
}

func (rrc readerReadCloser) Close() error {
	return nil // No resources to release
}

// ExampleWikiLoader demonstrates how a wiki file can be loaded and accessed by this package.
func ExampleWikiLoader() {
	// Dump lacks the XML prolog.
	o := bytes.NewReader([]byte(`<mediawiki xmlns="http://www.mediawiki.org/xml/export-0.11/" xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance" xsi:schemaLocation="http://www.mediawiki.org/xml/export-0.11/ http://www.mediawiki.org/xml/export-0.11.xsd" version="0.11" xml:lang="en">
  <siteinfo>
    <sitename>SomeWiki</sitename>
    <dbname>Somewiki</dbname>
    <base>http://somewiki.example.com/wiki/Main_Page</base>
    <generator>MediaWiki 1.39.6</generator>
    <case>first-letter</case>
    <namespaces>
      <namespace key="-2" case="first-letter">Media</namespace>
      <namespace key="-1" case="first-letter">Special</namespace>
      <namespace key="0" case="first-letter" />
      <namespace key="1" case="first-letter">Talk</namespace>
      <namespace key="2" case="first-letter">User</namespace>
      <namespace key="3" case="first-letter">User talk</namespace>
      <namespace key="4" case="first-letter">SomeWiki</namespace>
      <namespace key="5" case="first-letter">SomeWiki talk</namespace>
      <!-- ... -->
    </namespaces>
  </siteinfo>
  <page>
    <title>Grass</title>
    <ns>0</ns>
    <id>1234</id>
    <revision>
      <id>5678</id>
      <parentid>5677</parentid>
      <timestamp>2021-05-15T10:37:56Z</timestamp>
      <contributor>
        <username>TestUser</username>
        <id>1111</id>
      </contributor>
      <minor/>
      <comment>[bot] a comment by a bot</comment>
      <origin>5555</origin>
      <model>wikitext</model>
      <format>text/x-wiki</format>
      <text bytes="418" sha1="incorrect-sha-here" xml:space="preserve">Some text about grass could be here and might refer to [[Dirt]].

== Grass ==
{{#dpl:
| tablesortcol = 1
| mode=userformat
| category=Objects
| category=Grass
| namespace=
| include={{DPLPARM Object.include}}
| table={{DPLPARM Object.table}}
| tablerow={{DPLPARM Object.tablerow}}
| allowcachedresults=true
}}

See also other [[Objects]] and [[Grass (Tile)]].
[[Category:Grass]]</text>
      <sha1>incorrect sha</sha1>
    </revision>
  </page>
  <page>
    <title>Ice (Tile)</title>
    <ns>0</ns>
    <id>1235</id>
    <revision>
      <id>5680</id>
      <parentid>5679</parentid>
      <timestamp>2023-07-11T21:08:22Z</timestamp>
      <contributor>
        <username>SomeBot</username>
        <id>99999</id>
      </contributor>
      <comment>[bot] another comment by a bot.</comment>
      <origin>1012067</origin>
      <model>wikitext</model>
      <format>text/x-wiki</format>
      <!-- infobox taken verbatim from page 13693 revision 1044825 purely for testability --> 
      <text bytes="663" sha1="2suenxnyvxxnx6nubhqzqspccr7di03" xml:space="preserve">{{Infobox Object|List={{{1|}}}|GetValue={{{GetValue|}}}
| name          = Ice (Tile)
| actualname    = ice
| itemid        = 800, 6683, 6684, 6685, 6686
| objectclass   = Flooring
| primarytype   = Natural Tiles
| implemented   = 1.0
| immobile      = 
| walkable      = yes
| walkingspeed  = 100
| pickupable    = no
| mapcolor      = 179
| location      = On the [[Ice Islands]], and on top of the [[Dragonblaze Peaks]]. 10 sqm can also be found in the [[Rookgaard Academy]].
| notes         = Ice is intermixed with [[snow]], and therefore they sometimes looks very similar and you can leave your footprints in it. You move very quickly when walking on ice.
}}</text>
      <sha1>incorrect-sha</sha1>
    </revision>
  </page>
</mediawiki>
`))

	wl := NewWikiLoader(readerReadCloser{o})
	for !wl.Step() {
		// Currently not parseable; experimental code just prints it out on stdout.
		//
		// (But we can test that in this example.)
	}

	// The output contains "==> ", the item title, and the list of the item IDs.
	//
	// The output is experimental and should not be relied upon until a proper API is provided.

	// Output:
	// ==> "Ice (Tile)", [800 6683 6684 6685 6686]

}
