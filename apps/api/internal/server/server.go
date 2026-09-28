package server

import (
	"database/sql"
	"net/http"

	"github.com/ai-filmmaker/api/internal/config"
	"github.com/ai-filmmaker/api/internal/handlers"
	"github.com/ai-filmmaker/api/internal/repository"
	"github.com/ai-filmmaker/api/internal/service"
)

type Server struct {
	cfg            *config.Config
	db             *sql.DB
	router         *http.ServeMux
	projectHandler *handlers.ProjectHandler
}

func New(cfg *config.Config, db *sql.DB) *Server {
	var projectRepo repository.ProjectRepository
	if db != nil {
		projectRepo = repository.NewPostgresProjectRepository(db)
	}

	projectService := service.NewProjectService(projectRepo)
	projectHandler := handlers.NewProjectHandler(projectService)

	s := &Server{
		cfg:            cfg,
		db:             db,
		router:         http.NewServeMux(),
		projectHandler: projectHandler,
	}

	s.routes()
	return s
}

func (s *Server) routes() {
	s.router.HandleFunc("/health", handlers.HealthHandler)
	s.router.HandleFunc("/projects", s.projectHandler.ProjectDispatcher)
	s.router.HandleFunc("/projects/", s.projectHandler.ProjectDispatcher)
}

func (s *Server) Router() http.Handler {
	return s.corsMiddleware(s.router)
}

func (s *Server) corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PATCH, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func (s *Server) Start() error {
	addr := ":" + s.cfg.Port
	return http.ListenAndServe(addr, s.Router())
}
