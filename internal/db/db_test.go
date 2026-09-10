package db

import (
	"path/filepath"
	"testing"
)

func TestDB_FolderAndFileOperations(t *testing.T) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test.db")

	database, err := Open(dbPath)
	if err != nil {
		t.Fatalf("Failed to open test database: %v", err)
	}
	defer database.Close()

	// 1. Create root folder
	root, err := database.CreateFolder("Work", nil)
	if err != nil {
		t.Fatalf("CreateFolder root failed: %v", err)
	}

	// 2. Create subfolder
	sub, err := database.CreateFolder("Projects", &root.ID)
	if err != nil {
		t.Fatalf("CreateFolder sub failed: %v", err)
	}

	// 3. Test cycle prevention (cannot move root into its subfolder)
	err = database.MoveFolder(root.ID, &sub.ID)
	if err == nil {
		t.Fatalf("Expected error when moving parent into its child, got nil")
	}

	// 4. Create file in subfolder
	file, err := database.CreateFile(&sub.ID, "report.pdf", 1048576, "application/pdf", 1234, "file_xyz", "access_hash_abc", "dummy_sha256")
	if err != nil {
		t.Fatalf("CreateFile failed: %v", err)
	}

	// 5. Search file
	results, err := database.SearchFiles("report")
	if err != nil || len(results) != 1 {
		t.Fatalf("SearchFiles failed, expected 1 result, got %d (err: %v)", len(results), err)
	}
	if results[0].ID != file.ID {
		t.Fatalf("SearchFiles returned wrong file ID %s, expected %s", results[0].ID, file.ID)
	}

	// 6. Test share link creation
	share, err := database.CreateShareLink(file.ID, nil, nil, nil)
	if err != nil {
		t.Fatalf("CreateShareLink failed: %v", err)
	}

	loadedShare, err := database.GetShareLink(share.Token)
	if err != nil || loadedShare.FileID != file.ID {
		t.Fatalf("GetShareLink failed or returned mismatch")
	}

	// 7. Delete folder (cascading check)
	err = database.DeleteFolder(root.ID)
	if err != nil {
		t.Fatalf("DeleteFolder failed: %v", err)
	}

	_, err = database.GetFile(file.ID)
	// Foreign key set NULL or file remains if on delete set null
	// In schema: ON DELETE SET NULL for files(folder_id)
	if err != nil {
		t.Fatalf("File query after folder deletion failed: %v", err)
	}
}
