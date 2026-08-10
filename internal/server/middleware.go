package server

import (
	"net/http"
)

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
