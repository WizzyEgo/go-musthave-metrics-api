package storage

import "go-musthave-metrics-tpl/internal/model"

type Storage interface {
	UpdateGauge(name string, value float64)
	UpdateCounter(name string, value int64)
	UpdateBatch(metrics []model.Metrics) ([]model.Metrics, error)
	GetGauge(name string) (float64, bool)
	GetCounter(name string) (int64, bool)
	GetAll() (gauges map[string]float64, counters map[string]int64)
}
