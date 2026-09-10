package telegram

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/gotd/td/telegram/auth"
	"github.com/gotd/td/tg"
)

// ConsoleAuthenticator implements auth.UserAuthenticator using stdio prompts.
type ConsoleAuthenticator struct{}

func (ConsoleAuthenticator) Phone(ctx context.Context) (string, error) {
	fmt.Print("Enter your Telegram phone number (e.g. +628123456789): ")
	reader := bufio.NewReader(os.Stdin)
	phone, err := reader.ReadString('\n')
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(phone), nil
}

func (ConsoleAuthenticator) Password(ctx context.Context) (string, error) {
	fmt.Print("Enter your 2FA Cloud Password (if any): ")
	reader := bufio.NewReader(os.Stdin)
	pwd, err := reader.ReadString('\n')
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(pwd), nil
}

func (ConsoleAuthenticator) AcceptTermsOfService(ctx context.Context, tos tg.HelpTermsOfService) error {
	return nil
}

func (ConsoleAuthenticator) Code(ctx context.Context, sentCode *tg.AuthSentCode) (string, error) {
	fmt.Print("Enter the Telegram verification code received: ")
	reader := bufio.NewReader(os.Stdin)
	code, err := reader.ReadString('\n')
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(code), nil
}

func (ConsoleAuthenticator) SignUp(ctx context.Context) (auth.UserInfo, error) {
	return auth.UserInfo{}, fmt.Errorf("sign-up not supported: please register your account via official Telegram app first")
}

// AuthenticateInteractive performs terminal-based authentication and saves encrypted session.
func (m *ClientManager) AuthenticateInteractive(ctx context.Context) error {
	flow := auth.NewFlow(ConsoleAuthenticator{}, auth.SendCodeOptions{})

	status, err := m.client.Auth().Status(ctx)
	if err == nil && status.Authorized {
		fmt.Println("✓ Account is already authenticated in database session.")
		return m.EnsureStorageChannel(ctx)
	}

	if err := m.client.Auth().IfNecessary(ctx, flow); err != nil {
		return fmt.Errorf("interactive auth failed: %w", err)
	}

	fmt.Println("✓ MTProto authentication successful. Session encrypted and stored in SQLite.")
	return m.EnsureStorageChannel(ctx)
}

// EnsureStorageChannel verifies or creates the private Storage Channel "TeleDrive Vault".
func (m *ClientManager) EnsureStorageChannel(ctx context.Context) error {
	storedID, err := m.db.GetSetting("storage_channel_id")
	storedHash, errHash := m.db.GetSetting("storage_channel_hash")

	if err == nil && errHash == nil && storedID != "" && storedHash != "" {
		cID, _ := strconv.ParseInt(storedID, 10, 64)
		cHash, _ := strconv.ParseInt(storedHash, 10, 64)
		m.SetStorageChannel(cID, cHash)
		fmt.Printf("✓ Using existing Storage Channel (ID: %d)\n", cID)
		return nil
	}

	fmt.Println("Creating private Telegram Storage Channel 'TeleDrive Vault'...")
	updates, err := m.api.ChannelsCreateChannel(ctx, &tg.ChannelsCreateChannelRequest{
		Broadcast: true,
		Title:     "TeleDrive Vault",
		About:     "Private cloud storage object vault managed by TeleDrive. DO NOT delete or rename.",
	})
	if err != nil {
		return fmt.Errorf("failed to create storage channel: %w", err)
	}

	// Extract created channel information from updates
	var channelID int64
	var accessHash int64

	switch u := updates.(type) {
	case *tg.Updates:
		for _, chat := range u.Chats {
			if c, ok := chat.(*tg.Channel); ok {
				channelID = c.ID
				accessHash = c.AccessHash
				break
			}
		}
	}

	if channelID == 0 {
		return fmt.Errorf("could not extract created channel ID from Telegram response")
	}

	m.SetStorageChannel(channelID, accessHash)
	_ = m.db.SetSetting("storage_channel_id", strconv.FormatInt(channelID, 10))
	_ = m.db.SetSetting("storage_channel_hash", strconv.FormatInt(accessHash, 10))

	fmt.Printf("✓ Created private Storage Channel 'TeleDrive Vault' (ID: %d)\n", channelID)
	return nil
}
