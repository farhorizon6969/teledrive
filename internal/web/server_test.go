package web

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"path/filepath"
	"strings"
	"testing"

	"teledrive/internal/app"
	"teledrive/internal/db"
)

func TestWebServer_AuthAndFolderAPI(t *testing.T) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test.db")

	database, err := db.Open(dbPath)
	if err != nil {
		t.Fatalf("Open db failed: %v", err)
	}
	defer database.Close()

	cfg := &app.Config{
		Port:          "8080",
		DBPath:        dbPath,
		SecretKey:     "test-secret-key-32b",
		AdminPassword: "supersecretpassword",
	}

	server, err := NewServer(cfg, database, nil)
	if err != nil {
		t.Fatalf("NewServer failed: %v", err)
	}

	// 1. Unauthenticated request to / should redirect to /login
	req := httptest.NewRequest("GET", "/", nil)
	rec := httptest.NewRecorder()
	server.ServeHTTP(rec, req)

	if rec.Code != http.StatusSeeOther || rec.Header().Get("Location") != "/login" {
		t.Fatalf("Expected redirect to /login, got code %d, loc %s", rec.Code, rec.Header().Get("Location"))
	}

	// 2. Submit wrong password
	form := url.Values{"password": {"wrong"}}
	req = httptest.NewRequest("POST", "/login", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rec = httptest.NewRecorder()
	server.ServeHTTP(rec, req)

	if !strings.Contains(rec.Body.String(), "Invalid admin password") {
		t.Fatalf("Expected error message in login HTML, got body: %s", rec.Body.String())
	}

	// 3. Submit correct password
	form = url.Values{"password": {"supersecretpassword"}}
	req = httptest.NewRequest("POST", "/login", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rec = httptest.NewRecorder()
	server.ServeHTTP(rec, req)

	cookies := rec.Result().Cookies()
	var authCookie *http.Cookie
	for _, c := range cookies {
		if c.Name == "teledrive_session" {
			authCookie = c
			break
		}
	}
	if authCookie == nil || authCookie.Value != "authenticated" {
		t.Fatalf("Expected auth cookie to be set, got: %v", cookies)
	}

	// 4. Authenticated request to / should render dashboard HTML
	req = httptest.NewRequest("GET", "/", nil)
	req.AddCookie(authCookie)
	rec = httptest.NewRecorder()
	server.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), "TeleDrive") {
		t.Fatalf("Expected dashboard HTML 200 OK, got code %d", rec.Code)
	}

	// 5. Create folder via API
	folderPayload, _ := json.Marshal(map[string]any{"name": "Projects"})
	req = httptest.NewRequest("POST", "/api/folders", bytes.NewReader(folderPayload))
	req.Header.Set("Content-Type", "application/json")
	req.AddCookie(authCookie)
	rec = httptest.NewRecorder()
	server.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("Expected 201 Created for folder, got %d: %s", rec.Code, rec.Body.String())
	}

	// 6. List folders via API
	req = httptest.NewRequest("GET", "/api/folders", nil)
	req.AddCookie(authCookie)
	rec = httptest.NewRecorder()
	server.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), "Projects") {
		t.Fatalf("Expected folder list with 'Projects', got: %s", rec.Body.String())
	}

	// 7. Create a file & share link, then test GET /api/shares & DELETE /api/shares/{id}
	file, err := database.CreateFile(nil, "doc.pdf", 2048, "application/pdf", 11, "tg_1", "hash_1", "sha_1")
	if err != nil {
		t.Fatalf("CreateFile failed: %v", err)
	}

	sharePayload, _ := json.Marshal(map[string]any{"file_id": file.ID})
	req = httptest.NewRequest("POST", "/api/share", bytes.NewReader(sharePayload))
	req.Header.Set("Content-Type", "application/json")
	req.AddCookie(authCookie)
	rec = httptest.NewRecorder()
	server.ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("Expected 201 Created for share, got %d", rec.Code)
	}

	// GET /api/shares
	req = httptest.NewRequest("GET", "/api/shares", nil)
	req.AddCookie(authCookie)
	rec = httptest.NewRecorder()
	server.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), "doc.pdf") {
		t.Fatalf("Expected shares list containing 'doc.pdf', got %d: %s", rec.Code, rec.Body.String())
	}

	var shares []db.ShareLinkInfo
	_ = json.Unmarshal(rec.Body.Bytes(), &shares)
	if len(shares) == 0 {
		t.Fatalf("Expected at least 1 share in list")
	}

	// DELETE /api/shares/{id}
	req = httptest.NewRequest("DELETE", "/api/shares/"+shares[0].ID, nil)
	req.AddCookie(authCookie)
	rec = httptest.NewRecorder()
	server.ServeHTTP(rec, req)
	if rec.Code != http.StatusNoContent {
		t.Fatalf("Expected 204 No Content for delete share, got %d", rec.Code)
	}
}

