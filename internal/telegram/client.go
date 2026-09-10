package telegram

import (
	"context"
	"fmt"
	"sync"

	"github.com/gotd/td/telegram"
	"github.com/gotd/td/tg"
	"teledrive/internal/db"
)

type ClientManager struct {
	client    *telegram.Client
	api       *tg.Client
	db        *db.DB
	appID     int
	appHash   string
	secretKey string
	storage   *EncryptedSessionStorage

	mu                  sync.Mutex
	channelID           int64
	accessHash          int64
	configuredChannelID int64
}

// NewClientManager configures a new MTProto client instance with Safe Mode client headers.
func NewClientManager(database *db.DB, appID int, appHash, secretKey string) *ClientManager {
	storage := NewEncryptedSessionStorage(database, secretKey)

	// Organic client fingerprinting per ToS safe mode
	device := telegram.DeviceConfig{
		DeviceModel:    "PC 64bit",
		SystemVersion:  "Linux/x86_64",
		AppVersion:     "5.0.0",
		LangCode:       "en",
		SystemLangCode: "en",
	}

	client := telegram.NewClient(appID, appHash, telegram.Options{
		SessionStorage: storage,
		Device:         device,
	})

	return &ClientManager{
		client:    client,
		api:       tg.NewClient(client),
		db:        database,
		appID:     appID,
		appHash:   appHash,
		secretKey: secretKey,
		storage:   storage,
	}
}

// Client returns the underlying gotd telegram.Client.
func (m *ClientManager) Client() *telegram.Client {
	return m.client
}

// API returns the raw tg.Client for MTProto RPC calls.
func (m *ClientManager) API() *tg.Client {
	return m.api
}

// Run executes the client event loop until ctx is canceled.
func (m *ClientManager) Run(ctx context.Context, f func(ctx context.Context) error) error {
	return m.client.Run(ctx, f)
}

// SetStorageChannel caches the active storage channel credentials.
func (m *ClientManager) SetStorageChannel(channelID, accessHash int64) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.channelID = channelID
	m.accessHash = accessHash
}

// SetConfiguredChannelID sets an explicit storage channel ID from configuration or env var.
func (m *ClientManager) SetConfiguredChannelID(channelID int64) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.configuredChannelID = channelID
}

// SetDB updates the database reference (e.g. after snapshot restoration).
func (m *ClientManager) SetDB(database *db.DB) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.db = database
}

// StorageChannel returns the active storage channel channel_id and access_hash.
func (m *ClientManager) StorageChannel() (int64, int64, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.channelID == 0 {
		return 0, 0, fmt.Errorf("storage channel not initialized")
	}
	return m.channelID, m.accessHash, nil
}
