package repo

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"github.com/doug-martin/goqu/v9"
	_ "github.com/doug-martin/goqu/v9/dialect/sqlite3"
	"github.com/tamper000/multi-awg/internal/db"
	"github.com/tamper000/multi-awg/internal/models"
	"modernc.org/sqlite"
	sqlite3 "modernc.org/sqlite/lib"
)

var ErrPeerNameExists = errors.New("peer name already exists")

type PeerRepo struct {
	db *goqu.Database
}

func NewPeerRepo(db *sql.DB) *PeerRepo {
	return &PeerRepo{
		db: goqu.Dialect("sqlite3").DB(db),
	}
}

func (r *PeerRepo) CreatePeer(ctx context.Context, userID int64, name string) (*models.Peer, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("begin peer transaction: %w", err)
	}

	peer := models.Peer{
		UserID: userID,
		Name:   name,
	}

	err = tx.Wrap(func() error {
		result, err := tx.Insert(db.PeersTable).Rows(
			peer,
		).Executor().ExecContext(ctx)
		if err != nil {
			if isPeerNameExists(err) {
				return ErrPeerNameExists
			}
			return fmt.Errorf("create peer: %w", err)
		}

		id, err := result.LastInsertId()
		if err != nil {
			return fmt.Errorf("get peer id: %w", err)
		}

		peerName := fmt.Sprintf("%d.%d", userID, id)
		_, err = tx.Update(db.PeersTable).Set(
			goqu.Record{"peer_name": peerName},
		).Where(goqu.C("id").Eq(id)).Executor().ExecContext(ctx)
		if err != nil {
			return fmt.Errorf("set peer name: %w", err)
		}

		peer.ID = id
		peer.PeerName = peerName
		return nil
	})

	return &peer, err
}

func isPeerNameExists(err error) bool {
	var sqliteErr *sqlite.Error
	return errors.As(err, &sqliteErr) &&
		sqliteErr.Code() == sqlite3.SQLITE_CONSTRAINT_UNIQUE &&
		strings.Contains(sqliteErr.Error(), "peers.user_id, peers.name")
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

func (r *PeerRepo) DeleteByID(ctx context.Context, id int64) error {
	res, err := r.db.Delete(db.PeersTable).
		Where(goqu.C("id").Eq(id)).
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
