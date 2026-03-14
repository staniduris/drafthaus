package cli

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/drafthaus/drafthaus/internal/draft"
	"github.com/drafthaus/drafthaus/internal/server"
)

// ServeMulti serves all .draft files in dir, each mapped to a path prefix.
func ServeMulti(dir string, host string, port int) error {
	files, err := filepath.Glob(filepath.Join(dir, "*.draft"))
	if err != nil {
		return fmt.Errorf("scanning directory %q: %w", dir, err)
	}
	if len(files) == 0 {
		return fmt.Errorf("no .draft files found in %q", dir)
	}

	ms := server.NewMultiSiteServer(host, port)

	for _, f := range files {
		name := strings.TrimSuffix(filepath.Base(f), ".draft")
		store, err := draft.Open(f)
		if err != nil {
			return fmt.Errorf("open %q: %w", f, err)
		}
		if err := ms.AddSite(name, store); err != nil {
			store.Close()
			return fmt.Errorf("register site %q: %w", name, err)
		}
		log.Printf("Registered site: %s (%s)", name, f)
	}

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	errCh := make(chan error, 1)
	go func() {
		errCh <- ms.Start()
	}()

	select {
	case err := <-errCh:
		return err
	case <-quit:
		log.Println("Shutting down...")
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := ms.Shutdown(ctx); err != nil {
			log.Printf("warn: shutdown: %v", err)
		}
		return ms.Close()
	}
}

// ListSites lists all .draft files in dir with basic stats.
func ListSites(dir string) error {
	files, err := filepath.Glob(filepath.Join(dir, "*.draft"))
	if err != nil {
		return fmt.Errorf("scanning directory %q: %w", dir, err)
	}
	if len(files) == 0 {
		fmt.Printf("No .draft files found in %q\n", dir)
		return nil
	}

	fmt.Printf("Sites in %s:\n\n", dir)
	fmt.Printf("  %-24s  %-10s  %s\n", "NAME", "ENTITIES", "MODIFIED")
	fmt.Printf("  %-24s  %-10s  %s\n", "----", "--------", "--------")

	for _, f := range files {
		name := strings.TrimSuffix(filepath.Base(f), ".draft")

		info, err := os.Stat(f)
		if err != nil {
			fmt.Printf("  %-24s  (error reading file)\n", name)
			continue
		}

		entityCount := countEntities(f)
		fmt.Printf("  %-24s  %-10d  %s\n", name, entityCount, info.ModTime().Format("2006-01-02 15:04"))
	}

	return nil
}

func countEntities(path string) int {
	store, err := draft.Open(path)
	if err != nil {
		return -1
	}
	defer store.Close()

	types, err := store.ListTypes()
	if err != nil {
		return -1
	}

	total := 0
	for _, t := range types {
		_, count, err := store.ListEntities(t.ID, draft.ListOpts{Limit: 1})
		if err != nil {
			continue
		}
		total += count
	}
	return total
}
