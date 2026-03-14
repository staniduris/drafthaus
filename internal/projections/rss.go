package projections

import (
	"encoding/xml"
	"strings"
	"time"

	"github.com/drafthaus/drafthaus/internal/draft"
	"github.com/drafthaus/drafthaus/internal/graph"
)

// RSS is the top-level RSS 2.0 document.
type RSS struct {
	XMLName xml.Name   `xml:"rss"`
	Version string     `xml:"version,attr"`
	Channel RSSChannel `xml:"channel"`
}

// RSSChannel represents the RSS channel element.
type RSSChannel struct {
	Title       string    `xml:"title"`
	Link        string    `xml:"link"`
	Description string    `xml:"description"`
	LastBuild   string    `xml:"lastBuildDate"`
	Items       []RSSItem `xml:"item"`
}

// RSSItem represents a single RSS item.
type RSSItem struct {
	Title       string `xml:"title"`
	Link        string `xml:"link"`
	Description string `xml:"description"`
	PubDate     string `xml:"pubDate"`
	GUID        string `xml:"guid"`
}

// GenerateRSS produces an RSS 2.0 XML feed from the given entities.
func GenerateRSS(entities []*graph.ResolvedEntity, entityType *draft.EntityType, baseURL string) ([]byte, error) {
	listURL := baseURL
	if entityType.Routes != nil && entityType.Routes.List != "" {
		listURL = baseURL + entityType.Routes.List
	}

	items := make([]RSSItem, 0, len(entities))
	for _, re := range entities {
		items = append(items, buildRSSItem(re, baseURL))
	}

	feed := RSS{
		Version: "2.0",
		Channel: RSSChannel{
			Title:       entityType.Name,
			Link:        listURL,
			Description: entityType.Name + " feed",
			LastBuild:   time.Now().Format(time.RFC1123Z),
			Items:       items,
		},
	}

	raw, err := xml.MarshalIndent(feed, "", "  ")
	if err != nil {
		return nil, err
	}

	out := append([]byte("<?xml version=\"1.0\" encoding=\"utf-8\"?>\n"), raw...)
	return out, nil
}

func buildRSSItem(re *graph.ResolvedEntity, baseURL string) RSSItem {
	e := re.Entity

	title := stringField(e.Data, "title", "name", "")

	link := baseURL
	if re.Type.Routes != nil && re.Type.Routes.Detail != "" && e.Slug != "" {
		detail := strings.ReplaceAll(re.Type.Routes.Detail, "{slug}", e.Slug)
		detail = strings.ReplaceAll(detail, ":slug", e.Slug)
		link = baseURL + detail
	}

	description := stringField(e.Data, "excerpt", "description", "")
	if description == "" {
		description = firstBlockText(re.Blocks)
	}

	pubDate := pubDateFor(e)

	return RSSItem{
		Title:       title,
		Link:        link,
		Description: description,
		PubDate:     pubDate,
		GUID:        e.ID,
	}
}

// stringField returns the first non-empty string value from the data map
// for the given keys in order, or fallback if none match.
func stringField(data map[string]any, keys ...string) string {
	// last key is treated as fallback value (not a key lookup)
	if len(keys) < 2 {
		return ""
	}
	fallback := keys[len(keys)-1]
	lookupKeys := keys[:len(keys)-1]
	for _, k := range lookupKeys {
		if v, ok := data[k].(string); ok && v != "" {
			return v
		}
	}
	return fallback
}

func firstBlockText(blocks map[string][]*draft.Block) string {
	for _, fieldBlocks := range blocks {
		for _, b := range fieldBlocks {
			if b.Type == "paragraph" {
				if text, ok := b.Data["text"].(string); ok && text != "" {
					return text
				}
			}
		}
	}
	return ""
}

func pubDateFor(e *draft.Entity) string {
	if raw, ok := e.Data["published_at"].(string); ok && raw != "" {
		if t, err := time.Parse(time.RFC3339, raw); err == nil {
			return t.Format(time.RFC1123Z)
		}
	}
	if e.CreatedAt > 0 {
		return time.UnixMilli(e.CreatedAt).UTC().Format(time.RFC1123Z)
	}
	return time.Now().UTC().Format(time.RFC1123Z)
}
