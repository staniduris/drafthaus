package projections

import (
	"encoding/xml"
	"fmt"
	"strings"

	"github.com/drafthaus/drafthaus/internal/draft"
	"github.com/drafthaus/drafthaus/internal/graph"
)

type urlSet struct {
	XMLName xml.Name   `xml:"urlset"`
	XMLNS   string     `xml:"xmlns,attr"`
	URLs    []sitemapURL `xml:"url"`
}

type sitemapURL struct {
	Loc        string `xml:"loc"`
	ChangeFreq string `xml:"changefreq"`
	Priority   string `xml:"priority"`
}

// GenerateSitemap produces a sitemap XML document for all routes in the store.
func GenerateSitemap(store draft.Store, resolver *graph.Resolver, baseURL string) ([]byte, error) {
	baseURL = strings.TrimRight(baseURL, "/")

	urls := []sitemapURL{
		{Loc: baseURL + "/", ChangeFreq: "daily", Priority: "1.0"},
	}

	types, err := store.ListTypes()
	if err != nil {
		return nil, fmt.Errorf("list types: %w", err)
	}

	for _, t := range types {
		if t.Routes == nil {
			continue
		}

		if t.Routes.List != "" {
			urls = append(urls, sitemapURL{
				Loc:        baseURL + t.Routes.List,
				ChangeFreq: "weekly",
				Priority:   "0.7",
			})
		}

		if t.Routes.Detail != "" {
			entities, _, err := store.ListEntities(t.ID, draft.ListOpts{Status: "published", Limit: 1000})
			if err != nil {
				continue
			}
			for _, e := range entities {
				if e.Slug == "" {
					continue
				}
				loc := strings.ReplaceAll(t.Routes.Detail, "{slug}", e.Slug)
				loc = strings.ReplaceAll(loc, ":slug", e.Slug)
				urls = append(urls, sitemapURL{
					Loc:        baseURL + loc,
					ChangeFreq: "monthly",
					Priority:   "0.5",
				})
			}
		}
	}

	doc := urlSet{
		XMLNS: "http://www.sitemaps.org/schemas/sitemap/0.9",
		URLs:  urls,
	}

	raw, err := xml.MarshalIndent(doc, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("marshal sitemap: %w", err)
	}

	out := append([]byte("<?xml version=\"1.0\" encoding=\"utf-8\"?>\n"), raw...)
	return out, nil
}
