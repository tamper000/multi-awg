package handler

import (
	"context"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/tamper000/multi-awg/internal/auth"
	"github.com/tamper000/multi-awg/internal/models"
	"github.com/tamper000/multi-awg/internal/worker"
)

// Config — from env
type Config struct {
	JWTSecret  string
	TokenTTL   time.Duration
	MaxConfigs int
	StaticDir  string
}

type Handler struct {
	repo     UserRepo
	peers    PeerRepo
	sessions SessionRepo
	worker   WorkerClient
	config   Config
	tokens   *auth.TokenService
}

type UserRepo interface {
	CreateUser(ctx context.Context, username string, plainPassword string, role string, expiresAt *time.Time) error
	DeleteUser(ctx context.Context, id int64) error
	GetByUsername(ctx context.Context, username string) (*models.User, error)
	GetByID(ctx context.Context, id int64) (*models.User, error)
	GetBySubToken(ctx context.Context, token string) (*models.User, error)
	ListUsers(ctx context.Context) ([]models.User, error)
	UpdateExpiry(ctx context.Context, id int64, expiresAt *time.Time) error
	UpdatePassword(ctx context.Context, id int64, plainPassword string) error
	SetFrozen(ctx context.Context, id int64, frozen bool) error
}

type SessionRepo interface {
	CreateSession(ctx context.Context, userID int64, jti string, expiresAt time.Time) error
	DeleteSession(ctx context.Context, jti string) error
	SessionExists(ctx context.Context, jti string) (bool, error)
}

type PeerRepo interface {
	CreatePeer(ctx context.Context, userID int64, name string) (*models.Peer, error)
	DeleteByID(ctx context.Context, id int64) error
	GetByUserAndName(ctx context.Context, userID int64, name string) (*models.Peer, error)
	ListByUser(ctx context.Context, userID int64) ([]models.Peer, error)
}

type WorkerClient interface {
	CreatePeer(ctx context.Context, name string) (int, interface{}, error)
	DeletePeer(ctx context.Context, name string) (int, interface{}, error)
	DeletePeers(ctx context.Context, names []string) (int, interface{}, error)
	FreezePeers(ctx context.Context, names []string) (int, interface{}, error)
	GetPeerConfig(ctx context.Context, name string) (int, interface{}, error)
	GetPeersSub(ctx context.Context, peers []worker.SubPeer) (int, interface{}, error)
	GetPeerStats(ctx context.Context, name string) (int, interface{}, error)
	GetStats(ctx context.Context) (int, interface{}, error)
	Sync(ctx context.Context) (int, interface{}, error)
	UnfreezePeers(ctx context.Context, names []string) (int, interface{}, error)
}

func New(users UserRepo, sessions SessionRepo, peers PeerRepo, worker WorkerClient,
	cfg Config) *Handler {
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
		worker:   worker,
		config:   cfg,
		tokens:   auth.NewTokenService(cfg.JWTSecret, cfg.TokenTTL),
	}
}

func (h *Handler) Routes() http.Handler {
	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.Compress(5))

	r.Route("/api/auth", func(r chi.Router) {
		r.Post("/login", h.login)
		r.Post("/logout", h.logout)
	})

	r.Route("/api/admin", func(r chi.Router) {
		r.Use(h.authMiddleware)
		r.Use(h.adminMiddleware)
		r.Get("/users", h.listUsers)
		r.Post("/users", h.createUser)
		r.Get("/users/{id}", h.getUser)
		r.Delete("/users/{id}", h.deleteUser)
		r.Patch("/users/{id}", h.patchUser)
		r.Post("/sync", h.syncWorker)
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
		r.Get("/{token}/conf/{name}", h.subConf)
	})

	if h.config.StaticDir != "" {
		r.NotFound(h.serveSPA)
	}

	return r
}
