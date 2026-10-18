package storage

import (
	"database/sql"
	"errors"

	"github.com/pressly/goose/v3"

	"go-musthave-metrics-tpl/migrations"
)

func migrate(db *sql.DB) error {
	if db == nil {
		return errors.New("database is not configured")
	}

	goose.SetLogger(goose.NopLogger())
	goose.SetBaseFS(migrations.FS)
	if err := goose.SetDialect("postgres"); err != nil {
		return err
	}
	return goose.Up(db, ".")
}
