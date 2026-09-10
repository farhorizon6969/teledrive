package telegram

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/gotd/td/tg"
	"teledrive/internal/db"
)

// UploadSnapshot uploads a compressed SQLite database snapshot to the Storage Channel and pins it.
func (m *ClientManager) UploadSnapshot(ctx context.Context, gzPath string, limiter *SafeLimiter) (int, error) {
	fileInfo, err := os.Stat(gzPath)
	if err != nil {
		return 0, fmt.Errorf("stat snapshot: %w", err)
	}

	f, err := os.Open(gzPath)
	if err != nil {
		return 0, fmt.Errorf("open snapshot: %w", err)
	}
	defer f.Close()

	fileName := filepath.Base(gzPath)
	msgID, _, _, _, err := m.UploadFromReader(ctx, f, fileInfo.Size(), fileName, "application/gzip", limiter, nil)
	if err != nil {
		return 0, fmt.Errorf("upload snapshot document: %w", err)
	}

	channelID, accessHash, err := m.StorageChannel()
	if err != nil {
		return msgID, err
	}

	// Pin the latest backup message in the Storage Channel
	_, err = m.api.MessagesUpdatePinnedMessage(ctx, &tg.MessagesUpdatePinnedMessageRequest{
		Peer: &tg.InputPeerChannel{
			ChannelID:  channelID,
			AccessHash: accessHash,
		},
		ID: msgID,
	})
	if err != nil {
		fmt.Printf("Warning: failed to pin backup message (id %d): %v\n", msgID, err)
	}

	return msgID, nil
}

// RestoreLatestSnapshot searches the Storage Channel for the newest backup snapshot and restores it.
func (m *ClientManager) RestoreLatestSnapshot(ctx context.Context, destDBPath string) error {
	channelID, accessHash, err := m.StorageChannel()
	if err != nil {
		return err
	}

	peer := &tg.InputPeerChannel{
		ChannelID:  channelID,
		AccessHash: accessHash,
	}

	// Fetch recent messages from the Storage Channel
	history, err := m.api.MessagesGetHistory(ctx, &tg.MessagesGetHistoryRequest{
		Peer:  peer,
		Limit: 20,
	})
	if err != nil {
		return fmt.Errorf("get channel history: %w", err)
	}

	var targetDoc *tg.Document
	switch h := history.(type) {
	case *tg.MessagesChannelMessages:
		for _, message := range h.Messages {
			if msg, ok := message.(*tg.Message); ok {
				if media, ok := msg.Media.(*tg.MessageMediaDocument); ok {
					if doc, ok := media.Document.(*tg.Document); ok {
						for _, attr := range doc.Attributes {
							if fn, ok := attr.(*tg.DocumentAttributeFilename); ok {
								if strings.HasPrefix(fn.FileName, "teledrive-backup-") && strings.HasSuffix(fn.FileName, ".db.gz") {
									targetDoc = doc
									break
								}
							}
						}
					}
				}
			}
			if targetDoc != nil {
				break
			}
		}
	}

	if targetDoc == nil {
		return fmt.Errorf("no database backup snapshot found in Storage Channel")
	}

	tempGz := filepath.Join(os.TempDir(), "restored-temp.db.gz")
	defer os.Remove(tempGz)

	out, err := os.Create(tempGz)
	if err != nil {
		return fmt.Errorf("create temp download: %w", err)
	}
	defer out.Close()

	if err := m.DownloadFull(ctx, targetDoc.ID, targetDoc.AccessHash, out); err != nil {
		return fmt.Errorf("download backup snapshot: %w", err)
	}
	_ = out.Close()

	return db.RestoreFromGzip(tempGz, destDBPath)
}
