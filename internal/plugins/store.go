package plugins

// This file re-exports draft.PluginRecord for use by the plugin package
// and provides helper functions for loading plugins from the store.

import (
	"context"
	"fmt"

	"github.com/drafthaus/drafthaus/internal/draft"
)

// LoadFromStore loads all enabled plugins from the store into the runtime.
func LoadFromStore(ctx context.Context, rt *Runtime, store draft.Store) error {
	records, err := store.ListPlugins()
	if err != nil {
		return fmt.Errorf("list plugins: %w", err)
	}

	for _, rec := range records {
		if !rec.Enabled {
			continue
		}
		if _, err := rt.LoadPlugin(ctx, rec.Name, rec.Wasm); err != nil {
			// Log but don't fail — a broken plugin shouldn't prevent startup.
			fmt.Printf("warn: failed to load plugin %q: %v\n", rec.Name, err)
		}
	}
	return nil
}
