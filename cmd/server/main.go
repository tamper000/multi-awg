package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/tamper000/multi-awg/internal/db"
	"github.com/tamper000/multi-awg/internal/handler"
	userrepo "github.com/tamper000/multi-awg/internal/repo"
	"github.com/tamper000/multi-awg/internal/utils"
)

func main() {
	time.Local = time.UTC

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	database, err := db.Open(utils.GetEnv("DB_PATH", "app.db"))
	if err != nil {
		slog.Error("open db", "err", err)
		os.Exit(1)
	}
	defer database.Close()

	userRepo := userrepo.NewUserRepo(database)
	sessionRepo := userrepo.NewSessionRepo(database)
	peerRepo := userrepo.NewPeerRepo(database)
	if err := userRepo.EnsureDefaultAdmin(ctx); err != nil {
		slog.Error("ensure default admin", "err", err)
		os.Exit(1)
	}

	h := handler.New(userRepo, sessionRepo, peerRepo, handler.Config{
		WorkerURL:  utils.GetEnv("WORKER_URL", "http://127.0.0.1:9090"),
		Token:      utils.GetEnv("WORKER_TOKEN", "secret123"),
		JWTSecret:  utils.GetEnv("JWT_SECRET", "bd4dd1db95e05071a2182ff76bedf5162aebe6f17e506e3b39698cf106b7c371"),
		TokenTTL:   24 * time.Hour,
		MaxConfigs: utils.GetEnvInt("MAX_CONFIGS", 5),
		StaticDir:  utils.GetEnv("STATIC_DIR", "web/dist"),
	})

	addr := utils.GetEnv("ADDR", ":8080")
	srv := &http.Server{Addr: addr, Handler: h.Routes()}

	go func() {
		slog.Info("server listening", "addr", addr)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			slog.Error("listen", "err", err)
			os.Exit(1)
		}
	}()

	<-ctx.Done()
	slog.Info("shutting down")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		slog.Error("shutdown", "err", err)
	}
}
