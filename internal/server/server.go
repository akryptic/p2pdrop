package server

import (
	"net/http"

	"github.com/akryptic/p2pdrop/internal/config"
)

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

// Simple test handler
func (s *Server) handleDashboard(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("Welcome to P2P Drop Dashboard!"))
}

func (s *Server) handleSetupPage(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("Setup Wizard Page"))
}

func (s *Server) handleSaveConfig(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("Config Saved!"))
}

// Start boots up the underlying http.ListenAndServe on the specified port
func (s *Server) Start(addr string) error {
	return http.ListenAndServe(addr, s.router)
}

func (s *Server) RequireConfig(next http.HandlerFunc) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        // 1. If NOT configured AND trying to access restricted routes -> Redirect to /setup
        if !s.cfg.IsReady() && r.URL.Path != "/setup" && r.URL.Path != "/api/config" {
            http.Redirect(w, r, "/setup", http.StatusFound)
            return
        }
        // 2. Otherwise pass control to the intended page
        next(w, r)
    }
}