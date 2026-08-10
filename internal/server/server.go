package server

import (
	"embed"
	"fmt"
	"html/template"
	"net/http"

	"github.com/akryptic/p2pdrop/internal/config"
)

//go:embed templates/**
var templateFS embed.FS

type Server struct {
	cfg    *config.Config
	router *http.ServeMux
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
	fmt.Println("[ LOG ] Setup page hit")

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
	w.Write([]byte("Config Saved!"))
}

func (s *Server) handleDashboard(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("Welcome to P2P Drop Dashboard!"))
}

// Start boots up the underlying http.ListenAndServe on the specified port
func (s *Server) Start(addr string) error {
	return http.ListenAndServe(addr, s.router)
}
