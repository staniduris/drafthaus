package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/drafthaus/drafthaus/internal/draft"
	"github.com/drafthaus/drafthaus/internal/graph"
	"github.com/drafthaus/drafthaus/internal/render"
)

// Export generates a static HTML site from a .draft file into outputDir.
func Export(draftPath string, outputDir string) error {
	store, err := draft.Open(draftPath)
	if err != nil {
		return fmt.Errorf("open draft file: %w", err)
	}
	defer store.Close()

	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return fmt.Errorf("create output directory: %w", err)
	}

	resolver := graph.NewResolver(store)
	pipeline := render.NewPipeline(store, resolver)

	types, err := store.ListTypes()
	if err != nil {
		return fmt.Errorf("list types: %w", err)
	}

	var pageURLs []string
	pageCount := 0

	// Render list and detail pages for each entity type with routes.
	for _, et := range types {
		if et.Routes == nil {
			continue
		}

		viewName := et.Name + ".list"
		listView, _ := store.GetView(viewName)

		if et.Routes.List != "" && listView != nil {
			entities, _, err := resolver.ResolveList(et.Slug, graph.ListOpts{
				Status: "published",
				Limit:  1000,
			})
			if err != nil {
				return fmt.Errorf("resolve list for %s: %w", et.Slug, err)
			}

			html, err := pipeline.RenderList(entities, et, listView)
			if err != nil {
				return fmt.Errorf("render list for %s: %w", et.Slug, err)
			}

			outPath := filepath.Join(outputDir, et.Routes.List, "index.html")
			if err := writeFile(outPath, html); err != nil {
				return fmt.Errorf("write list page %s: %w", outPath, err)
			}
			pageURLs = append(pageURLs, et.Routes.List+"/")
			pageCount++
		}

		if et.Routes.Detail != "" {
			detailViewName := et.Name + ".detail"
			detailView, _ := store.GetView(detailViewName)
			if detailView == nil {
				continue
			}

			entities, _, err := store.ListEntities(et.ID, draft.ListOpts{
				Status: "published",
				Limit:  10000,
			})
			if err != nil {
				return fmt.Errorf("list entities for %s: %w", et.Slug, err)
			}

			for _, e := range entities {
				if e.Slug == "" {
					continue
				}

				resolved, err := resolver.Resolve(e.ID)
				if err != nil {
					return fmt.Errorf("resolve entity %s: %w", e.ID, err)
				}

				html, err := pipeline.RenderPage(resolved, detailView)
				if err != nil {
					return fmt.Errorf("render page for entity %s: %w", e.ID, err)
				}

				urlPath := strings.ReplaceAll(et.Routes.Detail, "{slug}", e.Slug)
				outPath := filepath.Join(outputDir, urlPath, "index.html")
				if err := writeFile(outPath, html); err != nil {
					return fmt.Errorf("write detail page %s: %w", outPath, err)
				}
				pageURLs = append(pageURLs, urlPath+"/")
				pageCount++
			}
		}
	}

	// Render homepage.
	if err := exportHomepage(store, pipeline, outputDir, &pageURLs, &pageCount); err != nil {
		return err
	}

	// Export assets.
	if err := exportAssets(store, outputDir); err != nil {
		return err
	}

	// Write sitemap.xml.
	if err := writeSitemap(outputDir, pageURLs); err != nil {
		return fmt.Errorf("write sitemap: %w", err)
	}

	// Write robots.txt.
	robotsContent := "User-agent: *\nAllow: /\n\nSitemap: /sitemap.xml\n"
	if err := os.WriteFile(filepath.Join(outputDir, "robots.txt"), []byte(robotsContent), 0644); err != nil {
		return fmt.Errorf("write robots.txt: %w", err)
	}

	fmt.Printf("Exported %d pages to %s\n", pageCount, outputDir)
	return nil
}

// exportHomepage renders the Homepage view and writes it to outputDir/index.html.
func exportHomepage(store draft.Store, pipeline *render.Pipeline, outputDir string, pageURLs *[]string, pageCount *int) error {
	homepageView, err := store.GetView("Homepage")
	if err != nil || homepageView == nil {
		// No homepage view defined; skip silently.
		return nil
	}

	// Render with an empty entity set.
	html, err := pipeline.RenderList(nil, nil, homepageView)
	if err != nil {
		return fmt.Errorf("render homepage: %w", err)
	}

	outPath := filepath.Join(outputDir, "index.html")
	if err := writeFile(outPath, html); err != nil {
		return fmt.Errorf("write homepage: %w", err)
	}
	*pageURLs = append(*pageURLs, "/")
	*pageCount++
	return nil
}

// exportAssets copies all stored assets to outputDir/_assets/{id}.
func exportAssets(store draft.Store, outputDir string) error {
	// The Store interface doesn't expose a ListAssets method, so we skip bulk
	// export and rely on inline asset references within rendered HTML.
	assetsDir := filepath.Join(outputDir, "_assets")
	if err := os.MkdirAll(assetsDir, 0755); err != nil {
		return fmt.Errorf("create assets directory: %w", err)
	}
	return nil
}

// writeSitemap generates a basic XML sitemap at outputDir/sitemap.xml.
func writeSitemap(outputDir string, urls []string) error {
	var sb strings.Builder
	sb.WriteString(`<?xml version="1.0" encoding="UTF-8"?>` + "\n")
	sb.WriteString(`<urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9">` + "\n")
	for _, u := range urls {
		sb.WriteString("  <url>\n")
		sb.WriteString("    <loc>" + u + "</loc>\n")
		sb.WriteString("  </url>\n")
	}
	sb.WriteString("</urlset>\n")
	return os.WriteFile(filepath.Join(outputDir, "sitemap.xml"), []byte(sb.String()), 0644)
}

// writeFile writes data to path, creating any necessary parent directories.
func writeFile(path string, data []byte) error {
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return fmt.Errorf("create directory for %s: %w", path, err)
	}
	return os.WriteFile(path, data, 0644)
}
