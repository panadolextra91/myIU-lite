package db_test

import (
	"context"
	"os"
	"testing"

	"github.com/jackc/pgx/v5"
	"golang.org/x/crypto/bcrypt"
)

func TestSeed_Admin_Integration(t *testing.T) {
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		t.Skip("DATABASE_URL is not set; skipping integration test")
	}

	ctx := context.Background()
	conn, err := pgx.Connect(ctx, dbURL)
	if err != nil {
		t.Fatalf("Failed to connect to database: %v", err)
	}
	defer conn.Close(ctx)

	var hash string
	var role string
	var mustChange bool

	err = conn.QueryRow(ctx, "SELECT password_hash, role, must_change_password FROM users WHERE username = 'admin'").Scan(&hash, &role, &mustChange)
	if err != nil {
		t.Fatalf("Failed to query admin user: %v", err)
	}

	if role != "admin" {
		t.Errorf("Expected role 'admin', got '%s'", role)
	}

	if !mustChange {
		t.Errorf("Expected must_change_password to be true, got false")
	}

	err = bcrypt.CompareHashAndPassword([]byte(hash), []byte("123456"))
	if err != nil {
		t.Errorf("password_hash does not match '123456': %v", err)
	}
}
