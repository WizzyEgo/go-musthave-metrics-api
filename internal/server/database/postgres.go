package database

import (
	"context"
	"database/sql"
	"errors"

	_ "github.com/jackc/pgx/v5/stdlib"

	"go-musthave-metrics-tpl/internal/retry"
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

func (c *Client) DB() *sql.DB {
	if c == nil {
		return nil
	}
	return c.db
}

func (c *Client) Ping(ctx context.Context) error {
	if c == nil || c.db == nil {
		return ErrNotConfigured
	}
	return retry.Do(ctx, func() error {
		err := c.db.PingContext(ctx)
		if retry.Postgres(err) {
			return retry.Retriable(err)
		}
		return err
	})
}

func (c *Client) Close() error {
	if c == nil || c.db == nil {
		return nil
	}
	return c.db.Close()
}
