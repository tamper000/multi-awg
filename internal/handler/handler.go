package handler

import (
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/tamper000/multi-awg/internal/auth"
	"github.com/tamper000/multi-awg/internal/repo"
	"github.com/tamper000/multi-awg/internal/worker"
)

// Config — from env
type Config struct {
	WorkerURL  string
	Token      string
	JWTSecret  string
	TokenTTL   time.Duration
	MaxConfigs int
	StaticDir  string
}

type Handler struct {
	repo     *repo.UserRepo
	peers    *repo.PeerRepo
	sessions *repo.SessionRepo
	worker   *worker.Client
	config   Config
	tokens   *auth.TokenService
}

func New(users *repo.UserRepo, sessions *repo.SessionRepo, peers *repo.PeerRepo, cfg Config) *Handler {
	if cfg.TokenTTL == 0 {
		cfg.TokenTTL = 24 * time.Hour
	}
	if cfg.MaxConfigs <= 0 {
		cfg.MaxConfigs = 5
	}
	return &Handler{
		repo:     users,
		peers:    peers,
		sessions: sessions,
		worker:   worker.New(cfg.WorkerURL, cfg.Token),
		config:   cfg,
		tokens:   auth.NewTokenService(cfg.JWTSecret, cfg.TokenTTL),
	}
}

func (h *Handler) Routes() http.Handler {
	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	r.Route("/api/auth", func(r chi.Router) {
		r.Post("/login", h.login)
		r.Post("/logout", h.logout)
	})

	r.Route("/api/admin", func(r chi.Router) {
		r.Use(h.authMiddleware)
		r.Use(h.adminMiddleware)
		r.Get("/users", h.listUsers)
		r.Post("/users", h.createUser)
		r.Get("/users/{username}", h.getUser)
		r.Delete("/users/{username}", h.deleteUser)
		r.Patch("/users/{username}", h.patchUser)
	})

	r.Route("/api/user", func(r chi.Router) {
		r.Use(h.authMiddleware)
		r.Get("/configs", h.listConfigs)
		r.Post("/configs", h.createConfig)
		r.Get("/configs/{name}", h.getConfig)
		r.Delete("/configs/{name}", h.deleteConfig)
		r.Get("/info", h.userInfo)
		r.Post("/password", h.password)
	})

	r.Route("/api/sub", func(r chi.Router) {
		r.Get("/{token}", h.subConfig)
		r.Get("/{token}/mihomo", h.subMihomo)
		r.Get("/{token}/conf", h.subConf)
	})

	if h.config.StaticDir != "" {
		r.NotFound(h.serveSPA)
	}

	return r
}
