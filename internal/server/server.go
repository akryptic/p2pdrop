package server

import (
	"context"
	"embed"
	"encoding/json"
	"html/template"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/akryptic/p2pdrop/internal/config"
)

//go:embed templates/**
var templateFS embed.FS

type Server struct {
	cfg    *config.Config
	router *http.ServeMux

	httpServer *http.Server
}

func NewServer(cfg *config.Config) *Server {
	s := &Server{
		cfg:    cfg,
		router: http.NewServeMux(),
	}

	// Register routes guarded by our middleware!
	s.router.HandleFunc("GET /setup", s.handleSetupPage)
	s.router.HandleFunc("POST /api/config", s.handleSaveConfig)
	s.router.HandleFunc("GET /", s.RequireConfig(s.handleDashboard))

	return s
}

func (s *Server) handleSetupPage(w http.ResponseWriter, r *http.Request) {
	// If already configured, redirect to dashbaord page
	if s.cfg.IsReady() {
		http.Redirect(w, r, "/", http.StatusFound)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")

	// parse base and setup templates
	tmpl := template.Must(template.ParseFS(templateFS, "templates/base.html", "templates/setup.html"))

	// 2. Execute "base" (the parent wrapper defined in base.html)
	if err := tmpl.ExecuteTemplate(w, "base", s.cfg.Get()); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func (s *Server) handleSaveConfig(w http.ResponseWriter, r *http.Request) {
	var req struct {
		DeviceName string `json:"device_name"`
		SaveDir    string `json:"save_dir"`
		UDPPort    uint16 `json:"udp_port"`
	}

	// 1. Validate JSON syntax
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Malformed JSON request body", http.StatusBadRequest)
		return
	}

	// 2. Validate Device Name
	deviceName := strings.TrimSpace(req.DeviceName)
	if deviceName == "" {
		http.Error(w, "Device name cannot be empty", http.StatusBadRequest)
		return
	}

	if len(deviceName) > 64 {
		http.Error(w, "Device name cannot exceed 64 characters", http.StatusBadRequest)
		return
	}

	// 3. Validate UDP Port (1024 - 65535)
	if req.UDPPort < 1024 {
		http.Error(w, "UDP port must be a non-privileged port between 1024 and 65535", http.StatusBadRequest)
		return
	}

	// 4. Validate & Normalize Save Directory
	saveDir := strings.TrimSpace(req.SaveDir)
	if saveDir == "" {
		http.Error(w, "Save directory cannot be empty", http.StatusBadRequest)
		return
	}

	// Handle tilde ~ expansion for home directory
	if strings.HasPrefix(saveDir, "~/") || saveDir == "~" {
		home, err := os.UserHomeDir()
		if err != nil {
			http.Error(w, "Could not resolve user home directory", http.StatusInternalServerError)
			return
		}
		saveDir = filepath.Join(home, saveDir[1:])
	}

	// Clean path to prevent relative path exploits (e.g. ./../../dir)
	cleanPath := filepath.Clean(saveDir)

	// 5. Persist sanitized values
	if err := s.cfg.WriteConfig(deviceName, cleanPath, req.UDPPort); err != nil {
		http.Error(w, "Failed to save configuration: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"status":"success"}`))
}

func (s *Server) handleDashboard(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")

	// Parse base.html and dashboard.html together
	tmpl := template.Must(template.ParseFS(templateFS, "templates/base.html", "templates/@.html"))

	if err := tmpl.ExecuteTemplate(w, "base", s.cfg.Get()); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

// Start boots up the underlying HTTP server
func (s *Server) Start(addr string) error {
	s.httpServer = &http.Server{
		Addr:    addr,
		Handler: s.router,
	}
	return s.httpServer.ListenAndServe()
}

// Shutdown gracefully stops the server without interrupting active transfers
func (s *Server) Shutdown(ctx context.Context) error {
	if s.httpServer != nil {
		return s.httpServer.Shutdown(ctx)
	}
	return nil
}
