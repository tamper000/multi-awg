package repo

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"fmt"

	"github.com/doug-martin/goqu/v9"
	_ "github.com/doug-martin/goqu/v9/dialect/sqlite3"
	"github.com/tamper000/multi-awg/internal/db"
	"github.com/tamper000/multi-awg/internal/models"
)

type PeerRepo struct {
	db *goqu.Database
}

func NewPeerRepo(db *sql.DB) *PeerRepo {
	return &PeerRepo{
		db: goqu.Dialect("sqlite3").DB(db),
	}
}

func generateSubToken() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

func (r *PeerRepo) CreatePeer(ctx context.Context, userID int64, name, peerName string) (string, error) {
	subToken, err := generateSubToken()
	if err != nil {
		return "", fmt.Errorf("generate sub token: %w", err)
	}
	_, err = r.db.Insert(db.PeersTable).Rows(
		goqu.Record{
			"user_id":   userID,
			"name":      name,
			"peer_name": peerName,
			"sub_token": subToken,
		},
	).Executor().ExecContext(ctx)
	if err != nil {
		return "", fmt.Errorf("create peer: %w", err)
	}
	return subToken, nil
}

func (r *PeerRepo) GetBySubToken(ctx context.Context, token string) (*models.Peer, error) {
	var p models.Peer
	found, err := r.db.From(db.PeersTable).
		Where(goqu.C("sub_token").Eq(token)).
		ScanStruct(&p)
	if err != nil {
		return nil, fmt.Errorf("get peer by sub token: %w", err)
	}
	if !found {
		return nil, ErrNotFound
	}
	return &p, nil
}

func (r *PeerRepo) ListByUser(ctx context.Context, userID int64) ([]models.Peer, error) {
	var peers []models.Peer
	if err := r.db.From(db.PeersTable).
		Where(goqu.C("user_id").Eq(userID)).
		Order(goqu.C("id").Asc()).
		ScanStructs(&peers); err != nil {
		return nil, fmt.Errorf("list peers: %w", err)
	}
	return peers, nil
}

func (r *PeerRepo) ListNamesByUser(ctx context.Context, userID int64) ([]string, error) {
	peers, err := r.ListByUser(ctx, userID)
	if err != nil {
		return nil, err
	}
	names := make([]string, 0, len(peers))
	for _, p := range peers {
		names = append(names, p.PeerName)
	}
	return names, nil
}

func (r *PeerRepo) GetByUserAndName(ctx context.Context, userID int64, name string) (*models.Peer, error) {
	var p models.Peer
	found, err := r.db.From(db.PeersTable).
		Where(goqu.C("user_id").Eq(userID), goqu.C("name").Eq(name)).
		ScanStruct(&p)
	if err != nil {
		return nil, fmt.Errorf("get peer: %w", err)
	}
	if !found {
		return nil, ErrNotFound
	}
	return &p, nil
}

func (r *PeerRepo) DeleteByName(ctx context.Context, userID int64, name string) error {
	res, err := r.db.Delete(db.PeersTable).
		Where(goqu.C("user_id").Eq(userID), goqu.C("name").Eq(name)).
		Executor().ExecContext(ctx)
	if err != nil {
		return fmt.Errorf("delete peer: %w", err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("delete peer rows affected: %w", err)
	}
	if n == 0 {
		return ErrNotFound
	}
	return nil
}
