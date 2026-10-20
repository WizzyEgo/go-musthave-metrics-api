package storage

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"go.uber.org/zap"

	"go-musthave-metrics-tpl/internal/model"
)

const dbTimeout = 3 * time.Second

var _ Storage = (*PostgresStorage)(nil)

type PostgresStorage struct {
	db  *sql.DB
	log *zap.Logger
}

func NewPostgresStorage(db *sql.DB, log *zap.Logger) (*PostgresStorage, error) {
	if log == nil {
		log = zap.NewNop()
	}
	if err := migrate(db); err != nil {
		return nil, err
	}
	return &PostgresStorage{db: db, log: log}, nil
}

func (s *PostgresStorage) UpdateGauge(name string, value float64) {
	ctx, cancel := context.WithTimeout(context.Background(), dbTimeout)
	defer cancel()

	_, err := s.db.ExecContext(ctx, `
		INSERT INTO metrics (id, mtype, value, delta)
		VALUES ($1, $2, $3, NULL)
		ON CONFLICT (id, mtype) DO UPDATE
		SET value = EXCLUDED.value, delta = NULL
	`, name, model.Gauge, value)
	if err != nil {
		s.log.Error("failed to update gauge", zap.String("id", name), zap.Error(err))
	}
}

func (s *PostgresStorage) UpdateCounter(name string, value int64) {
	ctx, cancel := context.WithTimeout(context.Background(), dbTimeout)
	defer cancel()

	_, err := s.db.ExecContext(ctx, `
		INSERT INTO metrics (id, mtype, delta, value)
		VALUES ($1, $2, $3, NULL)
		ON CONFLICT (id, mtype) DO UPDATE
		SET delta = COALESCE(metrics.delta, 0) + EXCLUDED.delta, value = NULL
	`, name, model.Counter, value)
	if err != nil {
		s.log.Error("failed to update counter", zap.String("id", name), zap.Error(err))
	}
}

func (s *PostgresStorage) GetGauge(name string) (float64, bool) {
	ctx, cancel := context.WithTimeout(context.Background(), dbTimeout)
	defer cancel()

	var value sql.NullFloat64
	err := s.db.QueryRowContext(ctx, `
		SELECT value FROM metrics WHERE id = $1 AND mtype = $2
	`, name, model.Gauge).Scan(&value)
	if err != nil {
		if !errors.Is(err, sql.ErrNoRows) {
			s.log.Error("failed to get gauge", zap.String("id", name), zap.Error(err))
		}
		return 0, false
	}
	if !value.Valid {
		return 0, false
	}
	return value.Float64, true
}

func (s *PostgresStorage) GetCounter(name string) (int64, bool) {
	ctx, cancel := context.WithTimeout(context.Background(), dbTimeout)
	defer cancel()

	var delta sql.NullInt64
	err := s.db.QueryRowContext(ctx, `
		SELECT delta FROM metrics WHERE id = $1 AND mtype = $2
	`, name, model.Counter).Scan(&delta)
	if err != nil {
		if !errors.Is(err, sql.ErrNoRows) {
			s.log.Error("failed to get counter", zap.String("id", name), zap.Error(err))
		}
		return 0, false
	}
	if !delta.Valid {
		return 0, false
	}
	return delta.Int64, true
}

func (s *PostgresStorage) GetAll() (map[string]float64, map[string]int64) {
	ctx, cancel := context.WithTimeout(context.Background(), dbTimeout)
	defer cancel()

	gauges := make(map[string]float64)
	counters := make(map[string]int64)

	rows, err := s.db.QueryContext(ctx, `SELECT id, mtype, value, delta FROM metrics`)
	if err != nil {
		s.log.Error("failed to list metrics", zap.Error(err))
		return gauges, counters
	}
	defer func() { _ = rows.Close() }()

	for rows.Next() {
		var (
			id    string
			mtype string
			value sql.NullFloat64
			delta sql.NullInt64
		)
		if err := rows.Scan(&id, &mtype, &value, &delta); err != nil {
			s.log.Error("failed to scan metric", zap.Error(err))
			continue
		}
		switch mtype {
		case model.Gauge:
			if value.Valid {
				gauges[id] = value.Float64
			}
		case model.Counter:
			if delta.Valid {
				counters[id] = delta.Int64
			}
		}
	}
	if err := rows.Err(); err != nil {
		s.log.Error("failed to iterate metrics", zap.Error(err))
	}
	return gauges, counters
}
