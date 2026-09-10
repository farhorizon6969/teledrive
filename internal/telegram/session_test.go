package telegram

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/gotd/td/session"
	"teledrive/internal/db"
)

func TestEncryptedSessionStorage(t *testing.T) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test.db")

	database, err := db.Open(dbPath)
	if err != nil {
		t.Fatalf("Failed to open db: %v", err)
	}
	defer database.Close()

	secret := "test-secret-key-for-session"
	storage := NewEncryptedSessionStorage(database, secret)

	ctx := context.Background()

	// 1. Initially, session should return session.ErrNotFound
	_, err = storage.LoadSession(ctx)
	if err != session.ErrNotFound {
		t.Fatalf("Expected ErrNotFound, got: %v", err)
	}

	// 2. Store session
	sampleData := []byte("mtproto-auth-key-binary-data-simulation")
	if err := storage.StoreSession(ctx, sampleData); err != nil {
		t.Fatalf("StoreSession failed: %v", err)
	}

	// 3. Load session and verify
	loaded, err := storage.LoadSession(ctx)
	if err != nil {
		t.Fatalf("LoadSession failed: %v", err)
	}
	if string(loaded) != string(sampleData) {
		t.Fatalf("Loaded session mismatch: got %q, want %q", string(loaded), string(sampleData))
	}

	// 4. Verify raw DB contains encrypted text, not raw session
	rawSetting, err := database.GetSetting("telegram_session")
	if err != nil {
		t.Fatalf("GetSetting failed: %v", err)
	}
	if rawSetting == string(sampleData) {
		t.Fatalf("Session was stored in plaintext, expected encrypted ciphertext!")
	}
}
