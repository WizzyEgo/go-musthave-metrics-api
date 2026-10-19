package storage

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"go.uber.org/zap"

	"go-musthave-metrics-tpl/internal/model"
	"go-musthave-metrics-tpl/internal/retry"
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
	err := s.withRetry(func(ctx context.Context) error {
		_, err := s.db.ExecContext(ctx, `
			INSERT INTO metrics (id, mtype, value, delta)
			VALUES ($1, $2, $3, NULL)
			ON CONFLICT (id, mtype) DO UPDATE
			SET value = EXCLUDED.value, delta = NULL
		`, name, model.Gauge, value)
		return err
	})
	if err != nil {
		s.log.Error("failed to update gauge", zap.String("id", name), zap.Error(err))
	}
}

func (s *PostgresStorage) UpdateCounter(name string, value int64) {
	err := s.withRetry(func(ctx context.Context) error {
		_, err := s.db.ExecContext(ctx, `
			INSERT INTO metrics (id, mtype, delta, value)
			VALUES ($1, $2, $3, NULL)
			ON CONFLICT (id, mtype) DO UPDATE
			SET delta = COALESCE(metrics.delta, 0) + EXCLUDED.delta, value = NULL
		`, name, model.Counter, value)
		return err
	})
	if err != nil {
		s.log.Error("failed to update counter", zap.String("id", name), zap.Error(err))
	}
}

const (
	upsertGaugeSQL = `
		INSERT INTO metrics (id, mtype, value, delta)
		VALUES ($1, $2, $3, NULL)
		ON CONFLICT (id, mtype) DO UPDATE
		SET value = EXCLUDED.value, delta = NULL
		RETURNING value`

	upsertCounterSQL = `
		INSERT INTO metrics (id, mtype, delta, value)
		VALUES ($1, $2, $3, NULL)
		ON CONFLICT (id, mtype) DO UPDATE
		SET delta = COALESCE(metrics.delta, 0) + EXCLUDED.delta, value = NULL
		RETURNING delta`
)

// UpdateBatch записывает набор метрик одной транзакцией.
// Инкремент counter выполняется в SQL, поэтому параллельные обновления не теряют дельту.
func (s *PostgresStorage) UpdateBatch(metrics []model.Metrics) ([]model.Metrics, error) {
	if err := validateMetrics(metrics); err != nil {
		return nil, err
	}
	if len(metrics) == 0 {
		return []model.Metrics{}, nil
	}

	var updated []model.Metrics
	err := s.withRetry(func(ctx context.Context) error {
		var err error
		updated, err = s.updateBatchTx(ctx, metrics)
		return err
	})
	if err != nil {
		s.log.Error("failed to update metrics batch", zap.Error(err))
		return nil, err
	}
	return updated, nil
}

func (s *PostgresStorage) updateBatchTx(ctx context.Context, metrics []model.Metrics) ([]model.Metrics, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback() }()

	gaugeStmt, err := tx.PrepareContext(ctx, upsertGaugeSQL)
	if err != nil {
		return nil, err
	}
	defer func() { _ = gaugeStmt.Close() }()

	counterStmt, err := tx.PrepareContext(ctx, upsertCounterSQL)
	if err != nil {
		return nil, err
	}
	defer func() { _ = counterStmt.Close() }()

	updated := make([]model.Metrics, len(metrics))
	for i, metric := range metrics {
		switch metric.MType {
		case model.Gauge:
			var value float64
			err = gaugeStmt.QueryRowContext(ctx, metric.ID, model.Gauge, *metric.Value).Scan(&value)
			if err != nil {
				return nil, err
			}
			updated[i] = metricFromStored(metric, value, 0)
		case model.Counter:
			var delta int64
			err = counterStmt.QueryRowContext(ctx, metric.ID, model.Counter, *metric.Delta).Scan(&delta)
			if err != nil {
				return nil, err
			}
			updated[i] = metricFromStored(metric, 0, delta)
		}
	}

	if err = tx.Commit(); err != nil {
		return nil, err
	}
	return updated, nil
}

func (s *PostgresStorage) GetGauge(name string) (float64, bool) {
	var value sql.NullFloat64
	err := s.withRetry(func(ctx context.Context) error {
		value = sql.NullFloat64{}
		return s.db.QueryRowContext(ctx, `
			SELECT value FROM metrics WHERE id = $1 AND mtype = $2
		`, name, model.Gauge).Scan(&value)
	})
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
	var delta sql.NullInt64
	err := s.withRetry(func(ctx context.Context) error {
		delta = sql.NullInt64{}
		return s.db.QueryRowContext(ctx, `
			SELECT delta FROM metrics WHERE id = $1 AND mtype = $2
		`, name, model.Counter).Scan(&delta)
	})
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
	gauges := map[string]float64{}
	counters := map[string]int64{}
	err := s.withRetry(func(ctx context.Context) error {
		gauges = map[string]float64{}
		counters = map[string]int64{}
		return s.loadAll(ctx, gauges, counters)
	})
	if err != nil {
		s.log.Error("failed to list metrics", zap.Error(err))
		return map[string]float64{}, map[string]int64{}
	}
	return gauges, counters
}

func (s *PostgresStorage) loadAll(ctx context.Context, gauges map[string]float64, counters map[string]int64) error {
	rows, err := s.db.QueryContext(ctx, `SELECT id, mtype, value, delta FROM metrics`)
	if err != nil {
		return err
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
			if retry.Postgres(err) {
				return err
			}
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
	return rows.Err()
}

func (s *PostgresStorage) withRetry(fn func(ctx context.Context) error) error {
	attempt := 0
	return retry.Do(context.Background(), func() error {
		attempt++
		ctx, cancel := context.WithTimeout(context.Background(), dbTimeout)
		defer cancel()

		err := fn(ctx)
		if err == nil || !retry.Postgres(err) {
			return err
		}
		s.log.Warn("retriable database error", zap.Int("attempt", attempt), zap.Error(err))
		return retry.Retriable(err)
	})
}
