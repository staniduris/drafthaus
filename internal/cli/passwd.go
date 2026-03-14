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

	// Delete existing user and recreate with new password
	db := store.DB()
	if _, err := db.Exec("DELETE FROM admin_users WHERE username = ?", username); err != nil {
		return fmt.Errorf("remove old user: %w", err)
	}

	if err := store.CreateAdminUser(username, newPassword); err != nil {
		return fmt.Errorf("create user: %w", err)
	}

	fmt.Printf("Password updated for %s\n", username)
	return nil
}
