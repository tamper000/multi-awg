package repo

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/doug-martin/goqu/v9"
	_ "github.com/doug-martin/goqu/v9/dialect/sqlite3"
	"github.com/tamper000/multi-awg/internal/db"
	"github.com/tamper000/multi-awg/internal/models"
	"github.com/tamper000/multi-awg/internal/utils/password"
)

const (
	RoleUser  = "user"
	RoleAdmin = "admin"
)

const (
	DefaultAdminUser = "admin"
	DefaultAdminPass = "admin"
)

var ErrNotFound = errors.New("not found")

type UserRepo struct {
	db *goqu.Database
}

func NewUserRepo(db *sql.DB) *UserRepo {
	return &UserRepo{
		db: goqu.Dialect("sqlite3").DB(db),
	}
}

func (r *UserRepo) CreateUser(ctx context.Context,
	username, plainPassword, role string,
	expiresAt *time.Time) error {

	hash, err := password.Hash(plainPassword)
	if err != nil {
		return fmt.Errorf("hash password: %w", err)
	}

	_, err = r.db.Insert(db.UsersTable).Rows(
		goqu.Record{
			"username":      username,
			"password_hash": hash,
			"role":          role,
			"expires_at":    expiresAt,
		},
	).Executor().ExecContext(ctx)
	if err != nil {
		return fmt.Errorf("insert user: %w", err)
	}

	return nil
}

func (r *UserRepo) GetByUsername(ctx context.Context, username string) (*models.User, error) {
	var u models.User
	found, err := r.db.From(db.UsersTable).
		Where(goqu.C("username").Eq(username)).
		ScanStruct(&u)
	if err != nil {
		return nil, fmt.Errorf("get user by username: %w", err)
	}
	if !found {
		return nil, ErrNotFound
	}
	return &u, nil
}

func (r *UserRepo) GetByID(ctx context.Context, id int64) (*models.User, error) {
	var u models.User
	found, err := r.db.From(db.UsersTable).
		Where(goqu.C("id").Eq(id)).
		ScanStruct(&u)
	if err != nil {
		return nil, fmt.Errorf("get user by id: %w", err)
	}
	if !found {
		return nil, ErrNotFound
	}
	return &u, nil
}

func (r *UserRepo) UpdatePassword(ctx context.Context, id int64, plainPassword string) error {
	hash, err := password.Hash(plainPassword)
	if err != nil {
		return fmt.Errorf("hash password: %w", err)
	}
	_, err = r.db.Update(db.UsersTable).Set(
		goqu.Record{"password_hash": hash},
	).Where(goqu.C("id").Eq(id)).Executor().ExecContext(ctx)
	if err != nil {
		return fmt.Errorf("update password: %w", err)
	}
	return nil
}

// SetFrozen ставит метку заморозки (true = подписка истекла и конфиги заморожены).
func (r *UserRepo) SetFrozen(ctx context.Context, id int64, frozen bool) error {
	_, err := r.db.Update(db.UsersTable).Set(
		goqu.Record{"frozen": frozen},
	).Where(goqu.C("id").Eq(id)).Executor().ExecContext(ctx)
	if err != nil {
		return fmt.Errorf("set frozen: %w", err)
	}
	return nil
}

func (r *UserRepo) ListUsers(ctx context.Context) ([]models.User, error) {
	var users []models.User
	if err := r.db.From(db.UsersTable).
		Order(goqu.C("id").Asc()).
		ScanStructs(&users); err != nil {
		return nil, fmt.Errorf("list users: %w", err)
	}
	return users, nil
}

func (r *UserRepo) DeleteUser(ctx context.Context, username string) error {
	res, err := r.db.Delete(db.UsersTable).
		Where(goqu.C("username").Eq(username)).
		Executor().ExecContext(ctx)
	if err != nil {
		return fmt.Errorf("delete user: %w", err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("delete user rows affected: %w", err)
	}
	if n == 0 {
		return ErrNotFound
	}
	return nil
}

// UpdateExpiry обновляет expires_at пользователя, возвращает ErrNotFound,
// если пользователь не найден.
func (r *UserRepo) UpdateExpiry(ctx context.Context, username string, expiresAt *time.Time) error {
	res, err := r.db.Update(db.UsersTable).Set(
		goqu.Record{"expires_at": expiresAt},
	).Where(goqu.C("username").Eq(username)).Executor().ExecContext(ctx)
	if err != nil {
		return fmt.Errorf("update expiry: %w", err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("update expiry rows affected: %w", err)
	}
	if n == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *UserRepo) EnsureDefaultAdmin(ctx context.Context) error {
	_, err := r.GetByUsername(ctx, DefaultAdminUser)
	if err == nil {
		return nil
	}
	if !errors.Is(err, ErrNotFound) {
		return err
	}
	return r.CreateUser(ctx, DefaultAdminUser, DefaultAdminPass, RoleAdmin, nil)
}
