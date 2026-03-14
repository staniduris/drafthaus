package cli

import (
	"fmt"
	"os"

	"github.com/drafthaus/drafthaus/internal/draft"
)

// Passwd changes the admin password for a .draft file.
func Passwd(path, username, newPassword string) error {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return fmt.Errorf("file not found: %s", path)
	}

	store, err := draft.Open(path)
	if err != nil {
		return fmt.Errorf("open draft file: %w", err)
	}
	defer store.Close()

	if err := store.UpdatePassword(username, newPassword); err != nil {
		return fmt.Errorf("update password: %w", err)
	}

	fmt.Printf("Password updated for %s\n", username)
	return nil
}
