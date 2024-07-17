package xmls

import (
	"encoding/xml"
	"fmt"
	"io"
	"log"
	"strconv"
	"strings"
)

// WikiLoader is a streaming XML reader of the mediawiki dump of the Tibia wiki.
//
// Wiki loading functionality is experimental and public API will change.
type WikiLoader struct {
	r          io.ReadSeekCloser
	xmlDecoder *xml.Decoder
}

func (e *WikiLoader) InnerXML(se xml.StartElement) (string, error) {
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

func (e *WikiLoader) XMLToken() (xml.Token, error) {
	t, err := e.xmlDecoder.Token()
	if err == nil {
		switch e := t.(type) {
		case xml.StartElement:
			for _, a := range e.Attr {
				log.Println(" ", a)
			}
		}
	}
	return t, err
}

func (e *WikiLoader) XMLRawToken() (xml.Token, error) {
	return e.xmlDecoder.RawToken()
}

// Namespace is a Go representation of the MediaWiki dump element 'namespace'.
//
// Experimental. Temporarily public.
type Namespace struct {
	xml.Name  `xml:"namespace,omitempty"`
	Key       string `xml:"key,attr,omitempty"`
	Case      string `xml:"case,attr,omitempty"`
	Namespace string `xml:",chardata"`
}

// SiteInfo is a Go representation of the MediaWiki dump element 'siteinfo'.
//
// Experimental. Temporarily public.
type SiteInfo struct {
	SiteName   string      `xml:"sitename,omitempty"`
	DBName     string      `xml:"dbname,omitempty"`
	Base       string      `xml:"base,omitempty"`
	Generator  string      `xml:"generator,omitempty"`
	Case       string      `xml:"case,omitempty"`
	Namespaces []Namespace `xml:"namespaces,omitempty"`
}

// Text is a Go representation of the MediaWiki dump element 'text'.
//
// Experimental. Temporarily public.
type Text struct {
	Bytes int    `xml:"bytes,omitempty"` // size
	SHA1  string `xml:"sha1,omitempty"`
	// xml:space="preserve"
	Content string `xml:",chardata"` // actual content
}

// CleanedContent attempts to provide just the bare minimum infobox content from the text.
//
// Some pages have "<noinclude>blob</noinclude>" before infobox, we need to trim that out first.
//
// Ideally we'd just extract the bits that are inside '{{Infobox .. }}' by parsing it all properly, but this is a good enough hack.
func (t *Text) CleanedContent() string {

	idx := strings.Index(t.Content, "{{Infobox ")
	if idx == -1 {
		return t.Content
	}
	return t.Content[idx:]
}

// Object turns Text into an Object, a map representation of the Object infobox contained in a wiki page.
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

// Object is a map[string]string representation of an infobox of type 'Object', representing a Tibia item / 'object'.
type Object map[string]string

// Name returns the item's name.
func (o Object) Name() string {
	if o == nil {
		return ""
	}
	return o["name"]
}

// Article returns the English language article used in front of the item's name, e.g 'an' for 'an apple'.
func (o Object) Article() string {
	if o == nil {
		return ""
	}
	return o["article"]
}

// ItemIDs processes and returns a slice of ints containing client IDs for the item described in the infobox.
//
// The version for which the clientIDs apply is not specified, but spotchecking, seems to be fine for 8.54 uses.
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

// BuyFrom processes a comma-separated list of names of who the item can be bought from.
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

// Pickupable returns whether an item can be picked up.
func (o Object) Pickupable() bool {
	if o == nil {
		return false
	}

	return o["pickupable"] == "yes"
}

// Revision is a Go representation of the MediaWiki dump element 'revision'.
//
// It is a single stored version of a page's content and metadata, essentially a 'commit' in version control systems.
//
// Experimental. Temporarily public.
type Revision struct {
	ID int `xml:"id,omitempty"`
	// parentid, timestamp, contributor, comment, origin, ...
	Model  string `xml:"model,omitempty"`
	Format string `xml:"format,omitempty"` // mime
	Text   []Text `xml:"text,omitempty"`
}

// Page is a Go representation of the MediaWiki dump element 'page'.
//
// It is a set of revisions of a page's content and metadata, along with its own metadata.
//
// Experimental. Temporarily public.
type Page struct {
	Namespace int        `xml:"ns,omitempty"`
	ID        int        `xml:"id,omitempty"`
	Title     string     `xml:"title,omitempty"`
	Revisions []Revision `xml:"revision,omitempty"`
}

// ReadSiteinfo reads the toplevel element 'siteinfo'.
func (e *WikiLoader) ReadSiteinfo(se *xml.StartElement) error {
	si := &SiteInfo{}
	err := e.DecodeElement(si, se)
	log.Printf("%+v", si)
	return err
}

// ReadPage reads the toplevel element 'page' describing a single wiki page.
//
// It should emit the pages into a channel, but currently does not do anything like that, and just prints them out.
//
// Something else should be receiving the pages, determining what they are, and emit the translated objects (Object, Creature, ...) if supported and desired.
func (e *WikiLoader) ReadPage(se *xml.StartElement) error {
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

// Step is a single step in processing the XML.
//
// It will read the startelement for the outer <mediawiki> tag, then individual startelements for its children <siteinfo>, <page>, etc.
//
// We can afford to parse one page at a time, as individual pages are not going to be that large; it's just the overall XML that is too large.
//
// Reads should continue as long as false is returned. NOTE: This may change into true, for consistency with Scanner.
func (e *WikiLoader) Step() bool {
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
	default:
		//log.Println("unhandled element type", se)
	}
	return false
}

// Complete reading and close the read stream etc.
func (e *WikiLoader) CloseStream() {
	e.r.Close()
}

// Skip skips processing from the current start element to the close element.
func (e *WikiLoader) Skip() error {
	if err := e.xmlDecoder.Skip(); err != nil {
		return err
	}
	return nil
}

// DecodeElement decodes the document from the current start element to the
// matching close element.
func (e *WikiLoader) DecodeElement(v interface{}, start *xml.StartElement) error {
	if err := e.xmlDecoder.DecodeElement(v, start); err != nil {
		return err
	}
	return nil
}

// NewWikiLoader prepares a streaming read, which can then be done in a loop by invoking the Step function.
func NewWikiLoader(r io.ReadSeekCloser) *WikiLoader {
	decoder := xml.NewDecoder(r)

	return &WikiLoader{r: r, xmlDecoder: decoder}
}
