package telegram

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/gotd/td/session"
	"teledrive/internal/crypto"
	"teledrive/internal/db"
)

// EncryptedSessionStorage implements session.Storage using SQLite and AES-256-GCM.
type EncryptedSessionStorage struct {
	db        *db.DB
	secretKey []byte
}

func NewEncryptedSessionStorage(database *db.DB, secretKey string) *EncryptedSessionStorage {
	return &EncryptedSessionStorage{
		db:        database,
		secretKey: crypto.DeriveKey(secretKey),
	}
}

func (s *EncryptedSessionStorage) LoadSession(ctx context.Context) ([]byte, error) {
	val, err := s.db.GetSetting("telegram_session")
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, session.ErrNotFound
		}
		return nil, fmt.Errorf("load session from db: %w", err)
	}

	decrypted, err := crypto.Decrypt(val, s.secretKey)
	if err != nil {
		return nil, fmt.Errorf("decrypt session: %w", err)
	}
	return decrypted, nil
}

func (s *EncryptedSessionStorage) StoreSession(ctx context.Context, data []byte) error {
	encrypted, err := crypto.Encrypt(data, s.secretKey)
	if err != nil {
		return fmt.Errorf("encrypt session: %w", err)
	}
	return s.db.SetSetting("telegram_session", encrypted)
}
