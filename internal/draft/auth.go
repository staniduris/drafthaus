package draft

import (
	"database/sql"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

// CreateAdminUser hashes the password and inserts a new admin user.
func (s *SQLiteStore) CreateAdminUser(username, password string) error {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("hash password: %w", err)
	}
	_, err = s.db.Exec(
		`INSERT INTO admin_users (id, username, password) VALUES (?, ?, ?)`,
		uuid.New().String(), username, string(hash),
	)
	if err != nil {
		return fmt.Errorf("create admin user: %w", err)
	}
	return nil
}

// ValidateCredentials checks a username/password pair. Returns true if valid.
func (s *SQLiteStore) ValidateCredentials(username, password string) (bool, error) {
	var hash string
	err := s.db.QueryRow(
		`SELECT password FROM admin_users WHERE username = ?`, username,
	).Scan(&hash)
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("query admin user: %w", err)
	}
	if err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)); err != nil {
		return false, nil
	}
	return true, nil
}

// HasAdminUsers returns true if at least one admin user exists.
func (s *SQLiteStore) HasAdminUsers() (bool, error) {
	var count int
	if err := s.db.QueryRow(`SELECT COUNT(*) FROM admin_users`).Scan(&count); err != nil {
		return false, fmt.Errorf("count admin users: %w", err)
	}
	return count > 0, nil
}
