package app

import (
	"os"
	"strconv"
)

type Config struct {
	Port             string
	DBPath           string
	SecretKey        string
	TelegramAppID    int
	TelegramAppHash  string
	StorageChannelID int64
	AdminPassword    string
}

// LoadConfig reads configuration from environment variables with safe defaults.
func LoadConfig() *Config {
	port := getEnv("TELEDRIVE_PORT", "8080")
	dbPath := getEnv("TELEDRIVE_DB_PATH", "teledrive.db")
	secretKey := getEnv("TELEDRIVE_SECRET_KEY", "teledrive-default-local-secret-32b")
	adminPass := getEnv("TELEDRIVE_ADMIN_PASSWORD", "admin123")

	appID, _ := strconv.Atoi(getEnv("TELEDRIVE_TG_APP_ID", "0"))
	appHash := getEnv("TELEDRIVE_TG_APP_HASH", "")
	channelID, _ := strconv.ParseInt(getEnv("TELEDRIVE_STORAGE_CHANNEL_ID", "0"), 10, 64)

	return &Config{
		Port:             port,
		DBPath:           dbPath,
		SecretKey:        secretKey,
		TelegramAppID:    appID,
		TelegramAppHash:  appHash,
		StorageChannelID: channelID,
		AdminPassword:    adminPass,
	}
}

func getEnv(key, defaultVal string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return defaultVal
}
