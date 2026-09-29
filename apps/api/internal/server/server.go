package server

import (
	"database/sql"
	"log"
	"net/http"
	"strings"

	"github.com/ai-filmmaker/api/internal/config"
	"github.com/ai-filmmaker/api/internal/handlers"
	"github.com/ai-filmmaker/api/internal/media"
	"github.com/ai-filmmaker/api/internal/repository"
	"github.com/ai-filmmaker/api/internal/service"
	"github.com/ai-filmmaker/api/internal/storage"
)

type Server struct {
	cfg             *config.Config
	router          *http.ServeMux
	projectHandler  *handlers.ProjectHandler
	mediaHandler    *handlers.MediaHandler
	timelineHandler *handlers.TimelineHandler
	storageHandler  *handlers.StorageHandler
}

// New creates a production server wired to real PostgreSQL, storage, and media processor instances.
func New(cfg *config.Config, db *sql.DB) *Server {
	var projectRepo repository.ProjectRepository
	var mediaRepo repository.MediaRepository
	var timelineRepo repository.TimelineRepository

	if db != nil {
		projectRepo = repository.NewPostgresProjectRepository(db)
		mediaRepo = repository.NewPostgresMediaRepository(db)
		timelineRepo = repository.NewPostgresTimelineRepository(db)
	}

	store, err := storage.NewFromConfig(cfg)
	if err != nil {
		log.Fatalf("Failed to initialize storage: %v", err)
	}

	processor, err := media.NewFFmpegProcessor(cfg.FFmpegPath, cfg.FFprobePath)
	if err != nil {
		log.Printf("[Server] Warning: FFmpeg processor not available: %v", err)
	}

	queue := media.NewMemoryQueue(100)

	return NewWithDependencies(cfg, projectRepo, mediaRepo, timelineRepo, store, processor, queue)
}

// NewWithRepository allows testing server routing with only project repository.
func NewWithRepository(cfg *config.Config, projectRepo repository.ProjectRepository) *Server {
	return NewWithDependencies(cfg, projectRepo, nil, nil, nil, nil, nil)
}

// NewWithDependencies allows explicit dependency injection for unit and integration testing.
func NewWithDependencies(
	cfg *config.Config,
	projectRepo repository.ProjectRepository,
	mediaRepo repository.MediaRepository,
	timelineRepo repository.TimelineRepository,
	store storage.Storage,
	processor media.Processor,
	queue media.Queue,
) *Server {
	projectService := service.NewProjectService(projectRepo)
	projectHandler := handlers.NewProjectHandler(projectService)

	mediaService := service.NewMediaService(mediaRepo, projectRepo, store, processor, queue)
	mediaHandler := handlers.NewMediaHandler(mediaService)

	var timelineHandler *handlers.TimelineHandler
	if timelineRepo != nil {
		timelineService := service.NewTimelineService(timelineRepo, projectRepo, mediaRepo)
		timelineHandler = handlers.NewTimelineHandler(timelineService)
	}

	var storageHandler *handlers.StorageHandler
	if store != nil {
		storageHandler = handlers.NewStorageHandler(store)
	}

	s := &Server{
		cfg:             cfg,
		router:          http.NewServeMux(),
		projectHandler:  projectHandler,
		mediaHandler:    mediaHandler,
		timelineHandler: timelineHandler,
		storageHandler:  storageHandler,
	}

	s.routes()
	return s
}

func (s *Server) routes() {
	s.router.HandleFunc("/health", handlers.HealthHandler)

	// Projects, Media, and Timeline multiplexer
	s.router.HandleFunc("/projects", s.projectHandler.ProjectDispatcher)
	s.router.HandleFunc("/projects/", func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "/media") {
			s.mediaHandler.MediaDispatcher(w, r)
		} else if strings.Contains(r.URL.Path, "/timeline") {
			if s.timelineHandler != nil {
				s.timelineHandler.TimelineDispatcher(w, r)
			} else {
				handlers.WriteError(w, http.StatusNotFound, "NOT_FOUND", "timeline handler unconfigured")
			}
		} else {
			s.projectHandler.ProjectDispatcher(w, r)
		}
	})

	// Storage direct handlers
	if s.storageHandler != nil {
		s.router.HandleFunc("/storage/upload/", s.storageHandler.Upload)
		s.router.HandleFunc("/storage/download/", s.storageHandler.Download)
	}
}

func (s *Server) Router() http.Handler {
	return s.corsMiddleware(s.router)
}

func (s *Server) corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS, HEAD")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, Content-Length")

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
