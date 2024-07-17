// Binary wikiloader reads an XML-formatted dump of tibiawiki, obtainable from its `Special:Statistics` page.
//
// This is experimental reader working with 2024-02-15 19:23:22 version of the dump.
package main

import (
	//"bufio"
	"encoding/xml"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"strconv"
	"strings"
)

var (
	wikiDumpPath = flag.String("wiki_dump_path", "tibiawiki_pages_current.xml", "Path to the tibiawiki dump.")
)

// Extractor is a streaming XML reader of the mediawiki dump of the Tibia wiki.
type Extractor struct {
	r          io.ReadSeekCloser
	xmlDecoder *xml.Decoder
}

func (e *Extractor) InnerXML(se xml.StartElement) (string, error) {
	type rawXML struct {
		Raw string `xml:",innerxml"`
	}
	var raw rawXML
	err := e.xmlDecoder.DecodeElement(&raw, &se)
	if err != nil {
		return "", err
	}
	out := raw.Raw

	return out, nil
}

func (e *Extractor) XMLToken() (xml.Token, error) {
	t, err := e.xmlDecoder.Token()
	if err == nil {
		switch e := t.(type) {
		case xml.StartElement:
			//e.currentTagPath.Append(e)
			//log.Println("<", e.Name)
			for _, a := range e.Attr {
				log.Println(" ", a)
			}
		}
		// TODO(ivucica): call 'pop' for EndElement? gets rid of need to call Swallowed.
	}
	return t, err
}

func (e *Extractor) XMLRawToken() (xml.Token, error) {
	return e.xmlDecoder.RawToken()
}

type Namespace struct {
	xml.Name  `xml:"namespace,omitempty"`
	Key       string `xml:"key,attr,omitempty"`
	Case      string `xml:"case,attr,omitempty"`
	Namespace string `xml:",chardata"`
}

type SiteInfo struct {
	SiteName   string      `xml:"sitename,omitempty"`
	DBName     string      `xml:"dbname,omitempty"`
	Base       string      `xml:"base,omitempty"`
	Generator  string      `xml:"generator,omitempty"`
	Case       string      `xml:"case,omitempty"`
	Namespaces []Namespace `xml:"namespaces,omitempty"`
}
type Text struct {
	Bytes int    `xml:"bytes,omitempty"` // size
	SHA1  string `xml:"sha1,omitempty"`
	// xml:space="preserve"
	Content string `xml:",chardata"` // actual content
}

func (t *Text) CleanedContent() string {
	// Some have "<noinclude>blob</noinclude>" before infobox, we should likely trim that out first.
	// Ideally we'd just extract the bits that are inside {{Infobox by parsing it all properly, but this is a hack.

	idx := strings.Index(t.Content, "{{Infobox ")
	if idx == -1 {
		return t.Content
	}
	return t.Content[idx:]
}

func (t *Text) Object() (Object, error) {
	o := make(Object)

	/*
		tr := strings.NewReader(t.Content)
		scan := bufio.NewScanner(tr)
		finish := scan.Scan() // skip the first line
		if finish {
			return nil // we failed to read anything beyond the initial infobox line
		}
		for scan.Scan() {
			ln := scan.Text()
			// we would get lines with '| ' but have to join with those that do not have this; this is tedious to parse
		}
	*/

	// we will pretend that we are allowed to split by '\n| '. This might break for more complicated pages, but otherwise we'd have to have a proper parser.
	elems := strings.Split(t.CleanedContent(), "\n| ")
	for _, elem := range elems {
		if !strings.Contains(elem, "=") {
			continue
		}
		if !strings.Contains(elem, " =") {
			if elem[0] == '{' {
				// {{Infobox Object line is expected to be like this
				continue
			}

			if elem[0] != '{' {
				panic(elem)
			} else {
				continue
			}
		}
		kv := strings.Split(elem, " =") // permit lines to end with just '=' without a whitespace
		k := strings.TrimRight(kv[0], " ")
		v := strings.TrimLeft(kv[1], " ")

		o[k] = v
	}
	return o, nil
}

type Object map[string]string

func (o Object) Name() string {
	if o == nil {
		return ""
	}
	return o["name"]
}

func (o Object) Article() string {
	if o == nil {
		return ""
	}
	return o["article"]
}

func (o Object) ItemIDs() []int {
	if o == nil {
		return nil
	}
	var ids []int

	for _, idS := range strings.Split(o["itemid"], ",") {
		idS = strings.Trim(idS, " ")
		if idS == "" {
			continue
		}
		id, err := strconv.Atoi(idS)
		if err != nil {
			log.Printf("item %q has bad itemid %q", o["name"], idS)
			continue
		}
		ids = append(ids, id)
	}

	return ids
}

func (o Object) BuyFrom() []string {
	if o == nil {
		return nil
	}
	var whoNames []string

	for _, whoName := range strings.Split(o["buyfrom"], ",") {
		whoName = strings.Trim(whoName, " ")
		whoNames = append(whoNames, whoName)
	}

	return whoNames
}

func (o Object) Pickupable() bool {
	if o == nil {
		return false
	}

	return o["pickupable"] == "yes"
}

type Revision struct {
	ID int `xml:"id,omitempty"`
	// parentid, timestamp, contributor, comment, origin, ...
	Model  string `xml:"model,omitempty"`
	Format string `xml:"format,omitempty"` // mime
	Text   []Text `xml:"text,omitempty"`
}
type Page struct {
	Namespace int        `xml:"ns,omitempty"`
	ID        int        `xml:"id,omitempty"`
	Title     string     `xml:"title,omitempty"`
	Revisions []Revision `xml:"revision,omitempty"`
}

func (e *Extractor) ReadSiteinfo(se *xml.StartElement) error {
	si := &SiteInfo{}
	err := e.DecodeElement(si, se)
	log.Printf("%+v", si)
	return err
}

func (e *Extractor) ReadPage(se *xml.StartElement) error {
	pg := &Page{}
	err := e.DecodeElement(pg, se)
	if err != nil {
		return err
	}
	if pg.Namespace != 0 {
		// File or something else
		return fmt.Errorf("only main namespace supported atm")
	}
	if strings.HasSuffix(pg.Title, "/Spoiler") {
		// ignore spoiler pages for now
		return fmt.Errorf("spoilers skipped")
	}
	if len(pg.Revisions) > 2 {
		log.Printf("too many revisions on %s", pg.Title)
		return fmt.Errorf("too many revisions on %s", pg.Title)
	}
	if len(pg.Revisions) == 0 {
		log.Printf("no revisions on %s", pg.Title)
		return fmt.Errorf("no revisions on %s", pg.Title)
	}
	if len(pg.Revisions[0].Text) == 0 {
		log.Printf("no text on %s revision %d", pg.Title, pg.Revisions[0].ID)
		return fmt.Errorf("no text on %s revision %d", pg.Title, pg.Revisions[0].ID)
	}

	text := pg.Revisions[0].Text[0]
	content := text.CleanedContent()

	idx := strings.Index(content, "{{Infobox ")
	if idx == -1 {
		// log.Printf("no infobox in %q", pg.Title)
		return fmt.Errorf("we seek only infoboxen, no such thing on %s", pg.Title)
	}

	switch {
	case strings.HasPrefix(content, "{{Infobox Object"):
		o, err := text.Object()
		if err != nil {
			return err
		}
		log.Printf(" ==> %q, %v", o.Name(), o.ItemIDs())

	case strings.HasPrefix(content, "{{Infobox Creature"):
		log.Printf("=> %q", pg.Title)
		log.Printf("-- Skipping npc")

	case strings.HasPrefix(content, "{{Infobox NPC"):
		log.Printf("=> %q", pg.Title)
		log.Printf("-- Skipping npc")

	case strings.HasPrefix(content, "{{Infobox Spell"):
		log.Printf("=> %q", pg.Title)
		log.Printf("-- Skipping spell")

	case strings.HasPrefix(content, "{{Infobox Effect"):
		log.Printf("=> %q", pg.Title)
		log.Printf("-- Skipping effect")

	case strings.HasPrefix(content, "{{Infobox Outfit"):
		log.Printf("=> %q", pg.Title)
		log.Printf("-- Skipping outfit")

	case strings.HasPrefix(content, "{{Infobox Corpse"):
		log.Printf("=> %q", pg.Title)
		log.Printf("-- Skipping corpse")

	case strings.HasPrefix(content, "{{Infobox Missile"):
		log.Printf("=> %q", pg.Title)
		log.Printf("-- Skipping missile")

	case strings.HasPrefix(content, "{{Infobox Geography"):
		log.Printf("=> %q", pg.Title)
		log.Printf("-- Skipping geography")

	case strings.HasPrefix(content, "{{Infobox World"):
		log.Printf("=> %q", pg.Title)
		log.Printf("-- Skipping world")

	case strings.HasPrefix(content, "{{Infobox Hunt"):
		log.Printf("=> %q", pg.Title)
		log.Printf("-- Skipping hunt")

	case strings.HasPrefix(content, "{{Infobox Building"):
		log.Printf("=> %q", pg.Title)
		log.Printf("-- Skipping building")

	case strings.HasPrefix(content, "{{Infobox Quest"):
		log.Printf("=> %q", pg.Title)
		log.Printf("-- Skipping quest")

	case strings.HasPrefix(content, "{{Infobox Key"):
		log.Printf("=> %q", pg.Title)
		log.Printf("-- Skipping key")

	case strings.HasPrefix(content, "{{Infobox House"):
		log.Printf("=> %q", pg.Title)
		log.Printf("-- Skipping house")

	case strings.HasPrefix(content, "{{Infobox Book"):
		log.Printf("=> %q", pg.Title)
		log.Printf("-- Skipping book")

	case strings.HasPrefix(content, "{{Infobox Mount"):
		log.Printf("=> %q", pg.Title)
		log.Printf("-- Skipping mount")

	case strings.HasPrefix(content, "{{Infobox Transcript"):
		log.Printf("=> %q", pg.Title)
		log.Printf("-- Skipping transcript")

	case strings.HasPrefix(content, "{{Infobox Achievement"):
		log.Printf("=> %q", pg.Title)
		log.Printf("-- Skipping achievement")

	case strings.HasPrefix(content, "{{Infobox Update"):
		log.Printf("=> %q", pg.Title)
		log.Printf("-- Skipping update")

	case strings.HasPrefix(content, "{{Infobox Cipsoft Member"):
		log.Printf("=> %q", pg.Title)
		log.Printf("-- Skipping cipsoft member")

	case strings.HasPrefix(content, "{{Infobox Street"):
		log.Printf("=> %q", pg.Title)
		log.Printf("-- Skipping street")

	case strings.HasPrefix(content, "{{Infobox Charm"):
		log.Printf("=> %q", pg.Title)
		log.Printf("-- Skipping charm")

	case strings.HasPrefix(content, "{{Infobox Imbuement"):
		log.Printf("=> %q", pg.Title)
		log.Printf("-- Skipping imbuement")

	case strings.HasPrefix(content, "{{Infobox Familiar"):
		log.Printf("=> %q", pg.Title)
		log.Printf("-- Skipping familiar")

	case strings.HasPrefix(content, "{{Infobox Tournament"):
		log.Printf("=> %q", pg.Title)
		log.Printf("-- Skipping tournament")

	case strings.HasPrefix(content, "{{Infobox Store Bundle"):
		log.Printf("=> %q", pg.Title)
		log.Printf("-- Skipping store bundle")

	case strings.HasPrefix(content, "{{Infobox Fansite"):
		//log.Printf("=> %q", pg.Title)
		//log.Printf("-- Skipping fansite")

	case strings.HasPrefix(content, "{{Infobox "):
		log.Printf("=> %q", pg.Title)
		components := strings.Split(content, "|")
		kind := strings.TrimPrefix(components[0], "{{Infobox ")
		kind = strings.TrimSuffix(kind, "\n ")
		log.Printf("-- Skipping unknown infobox type %q for %q", kind, pg.Title)

	case strings.HasPrefix(content, "#REDIRECT"):
		return nil
	case strings.HasPrefix(content, "#redirect"):
		return nil
	case strings.HasPrefix(content, "#Redirect"):
		return nil
	case strings.HasPrefix(content, "<!-- This list of NPCs is for"):
		return nil

	default:
		if content[0] != '{' {
			// Assume unparseable page
		}
		log.Printf("=> %q UNSUPPORTED", pg.Title)
		if len(content) < 50 {
			log.Printf("%q", content)
		} else {
			log.Printf("%q", content[0:50]+"...")
		}
	}
	return err
}

func (e *Extractor) Step() bool {
	// Read tokens from the XML document in a stream.
	t, err := e.XMLToken()
	if err != nil {
		log.Printf("failed to get an XML token: %v", err)
		return true
	}
	if t == nil {
		log.Printf("nil XML token")
		return true
	}

	switch se := t.(type) {
	case xml.StartElement:
		//log.Println("encountered " + se.Name.Local + " @ " + se.Name.Space)

		if se.Name.Space != "http://www.mediawiki.org/xml/export-0.11/" {
			// avoid checking the namespace below; we will only support this namespace anyway and skip any other startelements
			e.Skip()
			return false
		}
		switch se.Name.Local {
		case "mediawiki":
			// this is the outer shell, just read it in and continue
			return false
		case "siteinfo":
			e.ReadSiteinfo(&se)
		case "page":
			e.ReadPage(&se)
		default:
			log.Printf("don't know what to do with %s", se.Name.Local)
		}
	case xml.EndElement:
		if se.Name.Space == "http://www.mediawiki.org/xml/export-0.11" && se.Name.Local == "mediawiki" {
			e.CloseStream()
			return false
		}
		// ignore all other close elements
		e.Swallowed(se)
	default:
		//log.Println("unhandled element type", se)
	}
	return false
}

func (e *Extractor) CloseStream() {
	e.r.Close()
}

// Swallowed allows taghandlers to inform the Extractor that they have, in
// fact, eaten away some of the data. The end tag supplied must match the
// top of the tagpath stack or the connection will be closed.
//
// If an error is returned, the tag handler must return immediately.
func (e *Extractor) Swallowed(endElement xml.EndElement) error {
	// we have no tagpath impl right now so skip this
	return nil
}

// Skip skips processing from the current start element to the close element,
// and it pops the element from the tagpath.
func (e *Extractor) Skip() error {
	if err := e.xmlDecoder.Skip(); err != nil {
		return err
	}
	//return e.currentTagPath.Pop()
	return nil
}

// DecodeElement decodes the document from the current start element to the
// matching close element.
func (e *Extractor) DecodeElement(v interface{}, start *xml.StartElement) error {
	if err := e.xmlDecoder.DecodeElement(v, start); err != nil {
		return err
	}
	// return e.currentTagPath.Pop()
	return nil
}

// CurrentTagPath returns a copy of the current tag path.
//
// This can be useful to figure out information about the parent tags (e.g. what
// is the id of the iq tag).
func (e *Extractor) CurrentTagPath() interface{} {
	return "no support for tagpaths"
}

func NewExtractor(r io.ReadSeekCloser) *Extractor {
	decoder := xml.NewDecoder(r)

	return &Extractor{r: r, xmlDecoder: decoder}
}

func main() {
	f, err := os.Open(*wikiDumpPath)
	if err != nil {
		panic(err)
	}

	e := NewExtractor(f)
	for {
		if e.Step() {
			log.Print("breaking")
			break
		}
	}
}
