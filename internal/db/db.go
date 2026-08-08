package db

import (
	"database/sql"
	"embed"
	"fmt"

	"github.com/pressly/goose/v3"

	_ "modernc.org/sqlite"
)

//go:embed migrations/*.sql
var migrations embed.FS

const (
	UsersTable    = "users"
	SessionsTable = "sessions"
	PeersTable    = "peers"
)

// Open открывает SQLite, включает нужные прагмы и применяет миграции.
func Open(path string) (*sql.DB, error) {
	dsn := fmt.Sprintf("file:%s?_pragma=busy_timeout(5000)&_pragma=journal_mode(WAL)&_pragma=foreign_keys(1)", path)
	d, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, err
	}
	if err := d.Ping(); err != nil {
		d.Close()
		return nil, err
	}

	goose.SetBaseFS(migrations)
	if err := goose.SetDialect("sqlite3"); err != nil {
		d.Close()
		return nil, fmt.Errorf("set migration dialect: %w", err)
	}
	if err := goose.Up(d, "migrations"); err != nil {
		d.Close()
		return nil, fmt.Errorf("run migrations: %w", err)
	}

	return d, nil
}
