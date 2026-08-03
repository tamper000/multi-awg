package repo

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/doug-martin/goqu/v9"
	_ "github.com/doug-martin/goqu/v9/dialect/sqlite3"
	"github.com/tamper000/multi-awg/internal/db"
)

type SessionRepo struct {
	db *goqu.Database
}

func NewSessionRepo(db *sql.DB) *SessionRepo {
	return &SessionRepo{
		db: goqu.Dialect("sqlite3").DB(db),
	}
}

func (r *SessionRepo) CreateSession(ctx context.Context, userID int64, jti string, expiresAt time.Time) error {
	_, err := r.db.Insert(db.SessionsTable).Rows(
		goqu.Record{
			"user_id":    userID,
			"token_jti":  jti,
			"expires_at": expiresAt,
		},
	).Executor().ExecContext(ctx)
	if err != nil {
		return fmt.Errorf("create session: %w", err)
	}
	return nil
}

func (r *SessionRepo) DeleteSession(ctx context.Context, jti string) error {
	_, err := r.db.From(db.SessionsTable).
		Where(goqu.C("token_jti").Eq(jti)).
		Delete().Executor().ExecContext(ctx)
	if err != nil {
		return fmt.Errorf("delete session: %w", err)
	}
	return nil
}

func (r *SessionRepo) SessionExists(ctx context.Context, jti string) (bool, error) {
	var n int
	found, err := r.db.From(db.SessionsTable).
		Select(goqu.COUNT("*")).
		Where(goqu.C("token_jti").Eq(jti)).
		ScanVal(&n)
	if err != nil {
		return false, fmt.Errorf("session exists: %w", err)
	}
	return found && n > 0, nil
}
