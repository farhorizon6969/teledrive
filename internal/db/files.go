package db

import (
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"fmt"
	"time"
)

type Folder struct {
	ID        string    `json:"id"`
	ParentID  *string   `json:"parent_id"`
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type File struct {
	ID                 string    `json:"id"`
	FolderID           *string   `json:"folder_id"`
	Name               string    `json:"name"`
	Size               int64     `json:"size"`
	MimeType           string    `json:"mime_type"`
	TelegramMessageID  int       `json:"telegram_message_id"`
	TelegramFileID     string    `json:"telegram_file_id"`
	TelegramAccessHash string    `json:"telegram_access_hash"`
	SHA256             string    `json:"sha256"`
	CreatedAt          time.Time `json:"created_at"`
	UpdatedAt          time.Time `json:"updated_at"`
}

type UploadSession struct {
	ID             string    `json:"id"`
	FolderID       *string   `json:"folder_id"`
	Name           string    `json:"name"`
	Size           int64     `json:"size"`
	MimeType       string    `json:"mime_type"`
	TotalParts     int       `json:"total_parts"`
	UploadedParts  int       `json:"uploaded_parts"`
	TelegramFileID int64     `json:"telegram_file_id"`
	CreatedAt      time.Time `json:"created_at"`
}

type ShareLink struct {
	ID            string     `json:"id"`
	Token         string     `json:"token"`
	FileID        string     `json:"file_id"`
	PasswordHash  *string    `json:"password_hash"`
	ExpiresAt     *time.Time `json:"expires_at"`
	DownloadCount int        `json:"download_count"`
	MaxDownloads  *int       `json:"max_downloads"`
	CreatedAt     time.Time  `json:"created_at"`
}

// generateID produces a random 16-hex string ID.
func generateID() string {
	b := make([]byte, 8)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

// CreateFolder adds a new virtual folder.
func (d *DB) CreateFolder(name string, parentID *string) (*Folder, error) {
	id := generateID()
	_, err := d.Exec("INSERT INTO folders (id, parent_id, name) VALUES (?, ?, ?)", id, parentID, name)
	if err != nil {
		return nil, fmt.Errorf("create folder: %w", err)
	}
	return d.GetFolder(id)
}

// GetFolder retrieves a single folder by ID.
func (d *DB) GetFolder(id string) (*Folder, error) {
	row := d.QueryRow("SELECT id, parent_id, name, created_at, updated_at FROM folders WHERE id = ?", id)
	var f Folder
	if err := row.Scan(&f.ID, &f.ParentID, &f.Name, &f.CreatedAt, &f.UpdatedAt); err != nil {
		return nil, err
	}
	return &f, nil
}

// ListFolders lists subfolders inside parentID (or root if parentID is nil).
func (d *DB) ListFolders(parentID *string) ([]Folder, error) {
	var rows *sql.Rows
	var err error
	if parentID == nil {
		rows, err = d.Query("SELECT id, parent_id, name, created_at, updated_at FROM folders WHERE parent_id IS NULL ORDER BY name ASC")
	} else {
		rows, err = d.Query("SELECT id, parent_id, name, created_at, updated_at FROM folders WHERE parent_id = ? ORDER BY name ASC", *parentID)
	}
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var folders []Folder
	for rows.Next() {
		var f Folder
		if err := rows.Scan(&f.ID, &f.ParentID, &f.Name, &f.CreatedAt, &f.UpdatedAt); err != nil {
			return nil, err
		}
		folders = append(folders, f)
	}
	return folders, rows.Err()
}

// RenameFolder updates the name of a folder.
func (d *DB) RenameFolder(id, newName string) error {
	_, err := d.Exec("UPDATE folders SET name = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?", newName, id)
	return err
}

// MoveFolder moves a folder under a new parent (with cycle prevention check).
func (d *DB) MoveFolder(id string, newParentID *string) error {
	if newParentID != nil && *newParentID == id {
		return fmt.Errorf("cannot move folder inside itself")
	}
	// Check for ancestor cycle
	curr := newParentID
	for curr != nil {
		var p *string
		err := d.QueryRow("SELECT parent_id FROM folders WHERE id = ?", *curr).Scan(&p)
		if err != nil {
			break
		}
		if p != nil && *p == id {
			return fmt.Errorf("cannot move folder into its own descendant")
		}
		curr = p
	}

	_, err := d.Exec("UPDATE folders SET parent_id = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?", newParentID, id)
	return err
}

// DeleteFolder deletes a folder and cascades to subfolders and files.
func (d *DB) DeleteFolder(id string) error {
	_, err := d.Exec("DELETE FROM folders WHERE id = ?", id)
	return err
}

// CreateFile inserts a new file record linked to Telegram object metadata.
func (d *DB) CreateFile(folderID *string, name string, size int64, mimeType string, msgID int, fileID, accessHash, sha256 string) (*File, error) {
	id := generateID()
	_, err := d.Exec(`
		INSERT INTO files (id, folder_id, name, size, mime_type, telegram_message_id, telegram_file_id, telegram_access_hash, sha256)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, id, folderID, name, size, mimeType, msgID, fileID, accessHash, sha256)
	if err != nil {
		return nil, fmt.Errorf("create file record: %w", err)
	}
	return d.GetFile(id)
}

// GetFile retrieves a file by ID.
func (d *DB) GetFile(id string) (*File, error) {
	row := d.QueryRow(`
		SELECT id, folder_id, name, size, mime_type, telegram_message_id, telegram_file_id, telegram_access_hash, sha256, created_at, updated_at
		FROM files WHERE id = ?
	`, id)
	var f File
	if err := row.Scan(&f.ID, &f.FolderID, &f.Name, &f.Size, &f.MimeType, &f.TelegramMessageID, &f.TelegramFileID, &f.TelegramAccessHash, &f.SHA256, &f.CreatedAt, &f.UpdatedAt); err != nil {
		return nil, err
	}
	return &f, nil
}

// ListFiles returns all files within a specific folder (or root if folderID is nil).
func (d *DB) ListFiles(folderID *string) ([]File, error) {
	var rows *sql.Rows
	var err error
	if folderID == nil {
		rows, err = d.Query(`
			SELECT id, folder_id, name, size, mime_type, telegram_message_id, telegram_file_id, telegram_access_hash, sha256, created_at, updated_at
			FROM files WHERE folder_id IS NULL ORDER BY name ASC
		`)
	} else {
		rows, err = d.Query(`
			SELECT id, folder_id, name, size, mime_type, telegram_message_id, telegram_file_id, telegram_access_hash, sha256, created_at, updated_at
			FROM files WHERE folder_id = ? ORDER BY name ASC
		`, *folderID)
	}
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var files []File
	for rows.Next() {
		var f File
		if err := rows.Scan(&f.ID, &f.FolderID, &f.Name, &f.Size, &f.MimeType, &f.TelegramMessageID, &f.TelegramFileID, &f.TelegramAccessHash, &f.SHA256, &f.CreatedAt, &f.UpdatedAt); err != nil {
			return nil, err
		}
		files = append(files, f)
	}
	return files, rows.Err()
}

// RenameFile updates a file's name.
func (d *DB) RenameFile(id, newName string) error {
	_, err := d.Exec("UPDATE files SET name = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?", newName, id)
	return err
}

// MoveFile changes the parent folder of a file.
func (d *DB) MoveFile(id string, newFolderID *string) error {
	_, err := d.Exec("UPDATE files SET folder_id = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?", newFolderID, id)
	return err
}

// DeleteFile removes a file record.
func (d *DB) DeleteFile(id string) error {
	_, err := d.Exec("DELETE FROM files WHERE id = ?", id)
	return err
}

// SearchFiles finds files matching a name query.
func (d *DB) SearchFiles(query string) ([]File, error) {
	rows, err := d.Query(`
		SELECT id, folder_id, name, size, mime_type, telegram_message_id, telegram_file_id, telegram_access_hash, sha256, created_at, updated_at
		FROM files WHERE name LIKE ? ORDER BY name ASC LIMIT 50
	`, "%"+query+"%")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var files []File
	for rows.Next() {
		var f File
		if err := rows.Scan(&f.ID, &f.FolderID, &f.Name, &f.Size, &f.MimeType, &f.TelegramMessageID, &f.TelegramFileID, &f.TelegramAccessHash, &f.SHA256, &f.CreatedAt, &f.UpdatedAt); err != nil {
			return nil, err
		}
		files = append(files, f)
	}
	return files, rows.Err()
}

// CreateUploadSession initializes an in-flight upload session.
func (d *DB) CreateUploadSession(folderID *string, name string, size int64, mimeType string, totalParts int, tgFileID int64) (*UploadSession, error) {
	id := generateID()
	_, err := d.Exec(`
		INSERT INTO upload_sessions (id, folder_id, name, size, mime_type, total_parts, uploaded_parts, telegram_file_id)
		VALUES (?, ?, ?, ?, ?, ?, 0, ?)
	`, id, folderID, name, size, mimeType, totalParts, tgFileID)
	if err != nil {
		return nil, err
	}
	return &UploadSession{
		ID:             id,
		FolderID:       folderID,
		Name:           name,
		Size:           size,
		MimeType:       mimeType,
		TotalParts:     totalParts,
		UploadedParts:  0,
		TelegramFileID: tgFileID,
		CreatedAt:      time.Now(),
	}, nil
}

// IncrementUploadPart marks an additional part as received.
func (d *DB) IncrementUploadPart(id string) (int, error) {
	var count int
	err := d.QueryRow(`
		UPDATE upload_sessions SET uploaded_parts = uploaded_parts + 1 WHERE id = ? RETURNING uploaded_parts
	`, id).Scan(&count)
	return count, err
}

// GetUploadSession retrieves an active upload session.
func (d *DB) GetUploadSession(id string) (*UploadSession, error) {
	var s UploadSession
	err := d.QueryRow(`
		SELECT id, folder_id, name, size, mime_type, total_parts, uploaded_parts, telegram_file_id, created_at
		FROM upload_sessions WHERE id = ?
	`, id).Scan(&s.ID, &s.FolderID, &s.Name, &s.Size, &s.MimeType, &s.TotalParts, &s.UploadedParts, &s.TelegramFileID, &s.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &s, nil
}

// DeleteUploadSession cleans up an upload session.
func (d *DB) DeleteUploadSession(id string) error {
	_, err := d.Exec("DELETE FROM upload_sessions WHERE id = ?", id)
	return err
}

// CreateShareLink generates a public token for a file.
func (d *DB) CreateShareLink(fileID string, passwordHash *string, expiresAt *time.Time, maxDownloads *int) (*ShareLink, error) {
	id := generateID()
	token := generateID() + generateID() // 32 hex chars
	_, err := d.Exec(`
		INSERT INTO share_links (id, token, file_id, password_hash, expires_at, max_downloads)
		VALUES (?, ?, ?, ?, ?, ?)
	`, id, token, fileID, passwordHash, expiresAt, maxDownloads)
	if err != nil {
		return nil, err
	}
	return d.GetShareLink(token)
}

// GetShareLink finds a share link by its public token.
func (d *DB) GetShareLink(token string) (*ShareLink, error) {
	var sl ShareLink
	err := d.QueryRow(`
		SELECT id, token, file_id, password_hash, expires_at, download_count, max_downloads, created_at
		FROM share_links WHERE token = ?
	`, token).Scan(&sl.ID, &sl.Token, &sl.FileID, &sl.PasswordHash, &sl.ExpiresAt, &sl.DownloadCount, &sl.MaxDownloads, &sl.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &sl, nil
}

// IncrementShareDownload tracks download counts on a share link.
func (d *DB) IncrementShareDownload(token string) error {
	_, err := d.Exec("UPDATE share_links SET download_count = download_count + 1 WHERE token = ?", token)
	return err
}
