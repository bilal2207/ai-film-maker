package database

import (
	"database/sql"
	"fmt"
	"time"
)

type DB struct {
	conn *sql.DB
}

// Connect creates a database connection pool foundation.
// Note: Driver registration should occur when importing postgres driver (e.g. pgx/pq in future hops).
func Connect(databaseURL string) (*DB, error) {
	if databaseURL == "" {
		return nil, fmt.Errorf("database URL cannot be empty")
	}

	return &DB{}, nil
}

func (db *DB) Ping() error {
	if db.conn == nil {
		return nil
	}
	return db.conn.Ping()
}

func (db *DB) SetConnLimits(maxOpen, maxIdle int, maxLifetime time.Duration) {
	if db.conn != nil {
		db.conn.SetMaxOpenConns(maxOpen)
		db.conn.SetMaxIdleConns(maxIdle)
		db.conn.SetConnMaxLifetime(maxLifetime)
	}
}
