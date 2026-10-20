package storage

import (
	"context"

	"go-musthave-metrics-tpl/internal/model"
)

type Storage interface {
	UpdateGauge(ctx context.Context, name string, value float64) error
	UpdateCounter(ctx context.Context, name string, value int64) error
	UpdateBatch(ctx context.Context, metrics []model.Metrics) ([]model.Metrics, error)
	GetGauge(ctx context.Context, name string) (float64, bool)
	GetCounter(ctx context.Context, name string) (int64, bool)
	GetAll(ctx context.Context) (gauges map[string]float64, counters map[string]int64)
	Ping(ctx context.Context) error
}
