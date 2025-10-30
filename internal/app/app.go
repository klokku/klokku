package app

import (
	"net/http"
	"time"

	"github.com/gorilla/mux"
	"github.com/klokku/klokku/internal/config"
	"github.com/klokku/klokku/internal/database"
	"github.com/klokku/klokku/internal/rest"
	log "github.com/sirupsen/logrus"
)

// Application wires configuration, database, router, and server lifecycle.
type Application struct {
	cfg    config.Application
	router *mux.Router
	srv    *http.Server
}

// NewApplication constructs the full HTTP application, ready to Run().
func NewApplication() (*Application, error) {
	cfg, err := config.Load("./config/application.yaml")
	if err != nil {
		return nil, err
	}

	// DB + migrations
	db, err := database.Open(cfg)
	if err != nil {
		return nil, err
	}
	// db will be closed when server shuts down; defer not possible here, leave to process exit.
	if err := database.Migrate(cfg); err != nil {
		return nil, err
	}

	r := mux.NewRouter()

	// Build dependencies (services, handlers...)
	deps := BuildDependencies(db, cfg)

	// Middleware chain
	SetupMiddleware(r, deps, cfg)

	// Routes
	RegisterRoutes(r, deps, cfg)

	// Frontend
	if cfg.Frontend.Enabled {
		frontend := rest.NewFrontendHandler("frontend", "index.html")
		r.PathPrefix("/").Handler(frontend)
	}

	srv := &http.Server{
		Handler:      r,
		Addr:         ":8181",
		WriteTimeout: 15 * time.Second,
		ReadTimeout:  15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	return &Application{cfg: cfg, router: r, srv: srv}, nil
}

// Run starts the HTTP server and blocks.
func (a *Application) Run() error {
	log.Infof("Starting server on %s", a.srv.Addr)
	return a.srv.ListenAndServe()
}
