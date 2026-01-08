package server

import (
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	chimw "github.com/go-chi/chi/v5/middleware"
	"github.com/notifyd-eng/notifyd/internal/config"
	"github.com/notifyd-eng/notifyd/internal/middleware"
	"github.com/notifyd-eng/notifyd/internal/store"
)

type Server struct {
	cfg    *config.Config
	store  *store.Store
	router chi.Router
}

func New(cfg *config.Config, s *store.Store) *Server {
	srv := &Server{
		cfg:   cfg,
		store: s,
	}
	srv.setupRoutes()
	return srv
}

func (s *Server) setupRoutes() {
	r := chi.NewRouter()

	r.Use(chimw.RequestID)
	r.Use(chimw.RealIP)
	r.Use(middleware.Logger)
	r.Use(chimw.Recoverer)
	r.Use(chimw.Timeout(30 * time.Second))

	r.Get("/health", s.handleHealth)

	r.Route("/api/v1", func(r chi.Router) {
		if s.cfg.Server.APIKey != "" {
			r.Use(middleware.APIKey(s.cfg.Server.APIKey))
		}

		r.Route("/notifications", func(r chi.Router) {
			r.Post("/", s.handleCreate)
			r.Get("/", s.handleList)
			r.Get("/{id}", s.handleGet)
			r.Delete("/{id}", s.handleCancel)
		})

		r.Get("/stats", s.handleStats)
	})

	s.router = r
}

func (s *Server) ListenAndServe() error {
	httpSrv := &http.Server{
		Addr:         s.cfg.Server.Listen,
		Handler:      s.router,
		ReadTimeout:  s.cfg.Server.ReadTimeout,
		IdleTimeout:  s.cfg.Server.IdleTimeout,
		WriteTimeout: 60 * time.Second,
	}
	return httpSrv.ListenAndServe()
}
