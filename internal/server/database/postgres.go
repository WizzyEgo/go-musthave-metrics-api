package database

import (
	"context"
	"database/sql"
	"errors"

	_ "github.com/jackc/pgx/v5/stdlib"
)

var ErrNotConfigured = errors.New("database is not configured")

type Client struct {
	db *sql.DB
}

func Open(dsn string) (*Client, error) {
	if dsn == "" {
		return nil, ErrNotConfigured
	}
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		return nil, err
	}
	return &Client{db: db}, nil
}

func (c *Client) Ping(ctx context.Context) error {
	if c == nil || c.db == nil {
		return ErrNotConfigured
	}
	return c.db.PingContext(ctx)
}

func (c *Client) Close() error {
	if c == nil || c.db == nil {
		return nil
	}
	return c.db.Close()
}
