package service

import (
	"context"
	"log/slog"
	"time"

	"github.com/tamper000/multi-awg/internal/models"
	"github.com/tamper000/multi-awg/internal/repo"
)

type UserRepo interface {
	ListUsers(ctx context.Context) ([]models.User, error)
	SetFrozen(ctx context.Context, id int64, frozen bool) error
}

type PeerRepo interface {
	ListByUser(ctx context.Context, userID int64) ([]models.Peer, error)
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
		if user.ExpiresAt == nil {
			continue
		}

		if user.ExpiresAt.Before(now) && user.Role != repo.RoleAdmin {
			if user.Frozen {
				continue
			}

			peers, err := s.peerRepo.ListByUser(ctx, user.ID)
			if err != nil {
				slog.Error("get peers name", "userID", user.ID, "err", err)
				continue
			}

			names := make([]string, 0, len(peers))
			for _, peer := range peers {
				names = append(names, peer.PeerName)
			}
			if len(names) > 0 {
				_, _, err = s.worker.FreezePeers(ctx, names)
				if err != nil {
					slog.Error("freeze peers", "userID", user.ID, "err", err)
					continue
				}
				slog.Debug("froze peers", "userID", user.ID, "names", names)
			}

			if err := s.userRepo.SetFrozen(ctx, user.ID, true); err != nil {
				slog.Error("set frozen", "userID", user.ID, "err", err)
			}
		}
	}
}
