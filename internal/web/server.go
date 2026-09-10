package web

import (
	"embed"
	"fmt"
	"html/template"
	"io/fs"
	"net/http"
	"sync"
	"time"

	"teledrive/internal/app"
	"teledrive/internal/db"
	"teledrive/internal/telegram"
)

//go:embed templates/* static/*
var contentFS embed.FS

type Server struct {
	cfg       *app.Config
	db        *db.DB
	tg        *telegram.ClientManager
	limiter   *telegram.SafeLimiter
	mux       *http.ServeMux
	templates *template.Template

	// In-memory unlock cache for password-protected share tokens
	unlockedMu     sync.RWMutex
	unlockedTokens map[string]time.Time
}

func NewServer(cfg *app.Config, database *db.DB, tgManager *telegram.ClientManager) (*Server, error) {
	tmpl, err := template.ParseFS(contentFS, "templates/*.html")
	if err != nil {
		return nil, fmt.Errorf("parse embedded templates: %w", err)
	}

	s := &Server{
		cfg:            cfg,
		db:             database,
		tg:             tgManager,
		limiter:        telegram.NewSafeLimiter(),
		mux:            http.NewServeMux(),
		templates:      tmpl,
		unlockedTokens: make(map[string]time.Time),
	}

	s.routes()
	return s, nil
}

func (s *Server) routes() {
	// Embedded static assets
	staticFS, _ := fs.Sub(contentFS, "static")
	s.mux.Handle("GET /static/", http.StripPrefix("/static/", http.FileServer(http.FS(staticFS))))

	// Authentication
	s.mux.HandleFunc("GET /login", s.handleLoginPage)
	s.mux.HandleFunc("POST /login", s.handleLoginSubmit)
	s.mux.HandleFunc("GET /logout", s.handleLogout)

	// Protected Dashboard
	s.mux.HandleFunc("GET /", s.authMiddleware(s.handleDashboard))

	// Protected Drive API
	s.mux.HandleFunc("GET /api/folders", s.authMiddleware(s.handleListFolders))
	s.mux.HandleFunc("POST /api/folders", s.authMiddleware(s.handleCreateFolder))
	s.mux.HandleFunc("PUT /api/folders/{id}", s.authMiddleware(s.handleUpdateFolder))
	s.mux.HandleFunc("DELETE /api/folders/{id}", s.authMiddleware(s.handleDeleteFolder))

	s.mux.HandleFunc("GET /api/files", s.authMiddleware(s.handleListFiles))
	s.mux.HandleFunc("PUT /api/files/{id}", s.authMiddleware(s.handleUpdateFile))
	s.mux.HandleFunc("DELETE /api/files/{id}", s.authMiddleware(s.handleDeleteFile))
	s.mux.HandleFunc("GET /api/files/{id}/stream", s.authMiddleware(s.handleFileStream))
	s.mux.HandleFunc("GET /api/files/{id}/download", s.authMiddleware(s.handleFileDownload))

	// Resumable Chunked Upload API
	s.mux.HandleFunc("POST /api/upload/init", s.authMiddleware(s.handleUploadInit))
	s.mux.HandleFunc("POST /api/upload/chunk", s.authMiddleware(s.handleUploadChunk))
	s.mux.HandleFunc("POST /api/upload/complete", s.authMiddleware(s.handleUploadComplete))

	// Share Links
	s.mux.HandleFunc("POST /api/share", s.authMiddleware(s.handleCreateShare))
	s.mux.HandleFunc("GET /api/shares", s.authMiddleware(s.handleListShares))
	s.mux.HandleFunc("DELETE /api/shares/{id}", s.authMiddleware(s.handleDeleteShare))
	s.mux.HandleFunc("GET /s/{token}", s.handleShareLanding)
	s.mux.HandleFunc("POST /s/{token}/unlock", s.handleShareUnlock)
	s.mux.HandleFunc("GET /s/{token}/stream", s.handleShareStream)
	s.mux.HandleFunc("GET /s/{token}/download", s.handleShareDownload)
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.mux.ServeHTTP(w, r)
}

func (s *Server) authMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		cookie, err := r.Cookie("teledrive_session")
		if err != nil || cookie.Value != "authenticated" {
			if r.Header.Get("X-Requested-With") == "XMLHttpRequest" || r.URL.Path != "/" {
				http.Error(w, "Unauthorized", http.StatusUnauthorized)
				return
			}
			http.Redirect(w, r, "/login", http.StatusSeeOther)
			return
		}
		next(w, r)
	}
}

func (s *Server) handleLoginPage(w http.ResponseWriter, r *http.Request) {
	_ = s.templates.ExecuteTemplate(w, "login.html", nil)
}

func (s *Server) handleLoginSubmit(w http.ResponseWriter, r *http.Request) {
	pwd := r.FormValue("password")
	if pwd == s.cfg.AdminPassword {
		http.SetCookie(w, &http.Cookie{
			Name:     "teledrive_session",
			Value:    "authenticated",
			Path:     "/",
			HttpOnly: true,
			SameSite: http.SameSiteLaxMode,
			MaxAge:   86400 * 30, // 30 days
		})
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}

	_ = s.templates.ExecuteTemplate(w, "login.html", map[string]any{
		"Error": "Invalid admin password",
	})
}

func (s *Server) handleLogout(w http.ResponseWriter, r *http.Request) {
	http.SetCookie(w, &http.Cookie{
		Name:     "teledrive_session",
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		MaxAge:   -1,
	})
	http.Redirect(w, r, "/login", http.StatusSeeOther)
}

func (s *Server) handleDashboard(w http.ResponseWriter, r *http.Request) {
	_ = s.templates.ExecuteTemplate(w, "index.html", nil)
}
