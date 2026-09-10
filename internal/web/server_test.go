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
}
