package postgres

import (
	"database/sql"
	"fmt"
	"os"
	"strings"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
)

func OpenDB() (*sql.DB, error) {
	dsn := strings.TrimSpace(os.Getenv("POSTGRES_DSN"))
	if dsn == "" {
		return nil, fmt.Errorf("POSTGRES_DSN is required")
	}
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(10)
	db.SetMaxIdleConns(4)
	db.SetConnMaxLifetime(15 * time.Minute)
	if err := db.Ping(); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("postgres ping: %w", err)
	}
	return db, nil
}
