package main

import (
	"encoding/xml"
	"fmt"
	"net/http"
)

type SitemapChangeFreq int

const (
	SitemapChangeFreqUnspecified SitemapChangeFreq = 0
	SitemapChangeFreqAlways      SitemapChangeFreq = iota
	SitemapChangeFreqHourly
	SitemapChangeFreqDaily
	SitemapChangeFreqWeekly
	SitemapChangeFreqMonthly
	SitemapChangeFreqYearly
	SitemapChangeFreqNever
)

func (s SitemapChangeFreq) String() string {
	switch s {
	case SitemapChangeFreqUnspecified:
		return ""
	case SitemapChangeFreqAlways:
		return "always"
	case SitemapChangeFreqHourly:
		return "hourly"
	case SitemapChangeFreqDaily:
		return "daily"
	case SitemapChangeFreqWeekly:
		return "weekly"
	case SitemapChangeFreqMonthly:
		return "monthly"
	case SitemapChangeFreqYearly:
		return "yearly"
	case SitemapChangeFreqNever:
		return "never"
	}
	return "bad value"
}

type SitemapURLImage struct {
	// xml.Name would be 'http://www.google.com/schemas/sitemap-image/1.1 image'

	Loc string `xml:"image:loc"` // image is the namespace 'http://www.google.com/schemas/sitemap-image/1.1'
}

type SitemapURL struct {
	XMLName    xml.Name          `xml:"url"`
	Loc        string            `xml:"loc"`
	LastMod    string            `xml:"lastmod,omitempty"`
	ChangeFreq SitemapChangeFreq `xml:"changefreq,omitempty"`
	Priority   float32           `xml:"priority,omitempty"` // 0.0-1.0, default if unspecified is 0.5

	Image []SitemapURLImage `xml:"image:image,omitempty"`
}

type SitemapURLSet struct {
	XMLName    xml.Name     `xml:"http://www.sitemaps.org/schemas/sitemap/0.9 urlset"`
	XMLNSImage string       `xml:"xmlns:image,attr"`
	URL        []SitemapURL `xml:"url,omitempty"` // up to 50k entries
}

func (e *SitemapURLSet) Write(w http.ResponseWriter, r *http.Request) {
	e.XMLNSImage = "http://www.google.com/schemas/sitemap-image/1.1"

	w.Header().Set("Content-Type", "application/xml")

	fmt.Fprintf(w, "%s", xml.Header)
	enc := xml.NewEncoder(w)
	enc.Indent("", " ")
	err := enc.Encode(e)
	if err != nil {
		http.Error(w, "<error>could not encode sitemap</error>", http.StatusInternalServerError)
		return
	}
}

type SitemapIndexSitemap struct {
	XMLName xml.Name `xml:"http://www.sitemaps.org/schemas/sitemap/0.9 sitemap"`
	Loc     string   `xml:"loc"`
	LastMod string   `xml:"lastmod,omitempty"`
}

type SitemapIndex struct {
	XMLName xml.Name              `xml:"http://www.sitemaps.org/schemas/sitemap/0.9 sitemapindex"`
	Sitemap []SitemapIndexSitemap `xml:"sitemap"` // up to 50k entries
}

func (e *SitemapIndex) Write(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/xml")

	fmt.Fprintf(w, "%s", xml.Header)
	enc := xml.NewEncoder(w)
	enc.Indent("", " ")
	err := enc.Encode(e)
	if err != nil {
		http.Error(w, "<error>could not encode sitemap index</error>", http.StatusInternalServerError)
		return
	}
}
