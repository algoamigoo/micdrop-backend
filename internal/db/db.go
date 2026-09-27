package db

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5/stdlib"
	"github.com/jmoiron/sqlx"
)

// New opens a connection pool to Postgres and returns a *sqlx.DB.
func New(databaseURL string) (*sqlx.DB, error) {
	// 1. Parse the connection string into a config object
	cfg, err := pgxpool.ParseConfig(databaseURL)
	if err != nil {
		return nil, fmt.Errorf("parse db url: %w", err)
	}

	// 2. Configure the connection pool limits
	cfg.MaxConns = 20
	cfg.MinConns = 2
	cfg.MaxConnLifetime = time.Hour
	cfg.MaxConnIdleTime = 15 * time.Minute

	// 3. Create the pool
	pool, err := pgxpool.NewWithConfig(context.Background(), cfg)
	if err != nil {
		return nil, fmt.Errorf("connect db: %w", err)
	}

	// 4. Verify the database is actually reachable
	if err := pool.Ping(context.Background()); err != nil {
		pool.Close() // Clean up pool if ping fails
		return nil, fmt.Errorf("ping db: %w", err)
	}

	// 5. Wrap the pgx pool in sqlx using stdlib.OpenDBFromPool
	db := sqlx.NewDb(stdlib.OpenDBFromPool(pool), "pgx")
	return db, nil
}
