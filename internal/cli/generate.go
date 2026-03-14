package cli

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/drafthaus/drafthaus/internal/ai"
	"github.com/drafthaus/drafthaus/internal/draft"
)

// Generate creates a new .draft file from a natural language description.
func Generate(name, description string, cfg ai.Config) error {
	filename := name + ".draft"
	if _, err := os.Stat(filename); err == nil {
		return fmt.Errorf("file already exists: %s", filename)
	}

	provider, err := ai.NewProvider(cfg)
	if err != nil {
		return fmt.Errorf("create AI provider: %w", err)
	}
	if provider == nil {
		return fmt.Errorf("AI provider required for generate command (set --provider and --api-key)")
	}

	fmt.Printf("Generating site from description using %s...\n", provider.Name())

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	spec, err := ai.GenerateSite(ctx, provider, description)
	if err != nil {
		return fmt.Errorf("generate site spec: %w", err)
	}

	fmt.Printf("Creating %s with %d entity types...\n", filename, len(spec.EntityTypes))

	store, err := draft.Open(filename)
	if err != nil {
		return fmt.Errorf("create draft file: %w", err)
	}
	defer store.Close()

	if err := ai.ApplySiteSpec(store, spec); err != nil {
		return fmt.Errorf("apply site spec: %w", err)
	}

	if err := store.CreateAdminUser("admin", "admin"); err != nil {
		return fmt.Errorf("create admin: %w", err)
	}

	fmt.Printf("Created %s (%s)\n", filename, spec.SiteName)
	fmt.Println("Default admin: admin/admin — change this!")
	return nil
}
