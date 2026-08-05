package service

import (
	"context"
	"log/slog"
	"time"

	"github.com/tamper000/multi-awg/internal/models"
)

type UserRepo interface {
	ListUsers(ctx context.Context) ([]models.User, error)
}

type PeerRepo interface {
	ListNamesByUser(ctx context.Context, userID int64) ([]string, error)
}

type WorkerClient interface {
	FreezePeers(ctx context.Context, names []string) (int, interface{}, error)
}

type Service struct {
	userRepo UserRepo
	peerRepo PeerRepo
	worker   WorkerClient
}

func New(userRepo UserRepo, peerRepo PeerRepo, worker WorkerClient) *Service {
	return &Service{
		userRepo: userRepo,
		peerRepo: peerRepo,
		worker:   worker,
	}
}

func (s *Service) Start(ctx context.Context) {
	s.cleanup(ctx)

	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			s.cleanup(ctx)
		case <-ctx.Done():
			return
		}
	}
}

func (s *Service) cleanup(ctx context.Context) {
	users, err := s.userRepo.ListUsers(ctx)
	if err != nil {
		slog.Error("get users", "err", err)
		return
	}

	now := time.Now()
	for _, user := range users {
		if user.ExpiresAt == nil { // Admin
			continue
		}
		if user.ExpiresAt.Before(now) {
			names, err := s.peerRepo.ListNamesByUser(ctx, user.ID)
			if err != nil {
				slog.Error("get peers name", "userID", user.ID, "err", err)
				continue
			}

			_, _, err = s.worker.FreezePeers(ctx, names)
			if err != nil {
				slog.Error("freeze peers", "userID", user.ID, "err", err)
			}
		}
	}
}
