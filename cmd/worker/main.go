package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/docker/docker/client"
)

const amneziaName = "amneziawg"

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		panic(err)
	}
	defer cli.Close()

	pwd, err := os.Getwd()
	if err != nil {
		panic(err)
	}

	awgConfDir := filepath.Join(pwd, "amnezia-config")
	srv := &http.Server{
		Addr: ":9090",
		Handler: NewServer(cli, Config{
			ContainerName:  amneziaName,
			ConfDir:        awgConfDir,
			Token:          os.Getenv("AUTH_TOKEN"),
			ServerEndpoint: os.Getenv("SERVER_ENDPOINT"),
			MihomoTemplate: os.Getenv("MIHOMO_TEMPLATE"),
		}).Handler(),
	}

	go func() {
		slog.Info("worker API listening", "addr", ":9090")
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			slog.Error("worker API error", "err", err)
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
