package server

import (
	"net/http"

	"github.com/ai-filmmaker/api/internal/config"
	"github.com/ai-filmmaker/api/internal/handlers"
)

type Server struct {
	cfg    *config.Config
	router *http.ServeMux
}

func New(cfg *config.Config) *Server {
	s := &Server{
		cfg:    cfg,
		router: http.NewServeMux(),
	}

	s.routes()
	return s
}

func (s *Server) routes() {
	s.router.HandleFunc("/health", handlers.HealthHandler)
}

func (s *Server) Router() http.Handler {
	return s.router
}

func (s *Server) Start() error {
	addr := ":" + s.cfg.Port
	return http.ListenAndServe(addr, s.router)
}
