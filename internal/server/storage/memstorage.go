package storage

import (
	"sync"

	"go-musthave-metrics-tpl/internal/model"
)

var _ Storage = (*MemStorage)(nil)

type MemStorage struct {
	mu       sync.RWMutex
	gauges   map[string]float64
	counters map[string]int64
}

func NewMemStorage() *MemStorage {
	return &MemStorage{
		gauges:   make(map[string]float64),
		counters: make(map[string]int64),
	}
}

func (ms *MemStorage) UpdateGauge(name string, value float64) {
	ms.mu.Lock()
	defer ms.mu.Unlock()
	ms.gauges[name] = value
}

func (ms *MemStorage) UpdateCounter(name string, value int64) {
	ms.mu.Lock()
	defer ms.mu.Unlock()
	ms.counters[name] += value
}

func (ms *MemStorage) SetCounter(name string, value int64) {
	ms.mu.Lock()
	defer ms.mu.Unlock()
	ms.counters[name] = value
}

func (ms *MemStorage) GetGauge(name string) (float64, bool) {
	ms.mu.RLock()
	defer ms.mu.RUnlock()
	value, ok := ms.gauges[name]
	return value, ok
}

func (ms *MemStorage) GetCounter(name string) (int64, bool) {
	ms.mu.RLock()
	defer ms.mu.RUnlock()
	value, ok := ms.counters[name]
	return value, ok
}

func (ms *MemStorage) GetAll() (map[string]float64, map[string]int64) {
	ms.mu.RLock()
	defer ms.mu.RUnlock()

	gauges := make(map[string]float64, len(ms.gauges))
	for name, value := range ms.gauges {
		gauges[name] = value
	}

	counters := make(map[string]int64, len(ms.counters))
	for name, value := range ms.counters {
		counters[name] = value
	}

	return gauges, counters
}

// UpdateBatch применяет список метрик под одним замком, чтобы чтение не увидело часть батча.
func (ms *MemStorage) UpdateBatch(metrics []model.Metrics) ([]model.Metrics, error) {
	ms.mu.Lock()
	defer ms.mu.Unlock()

	if err := validateMetrics(metrics); err != nil {
		return nil, err
	}

	updated := make([]model.Metrics, len(metrics))
	for i, metric := range metrics {
		switch metric.MType {
		case model.Gauge:
			ms.gauges[metric.ID] = *metric.Value
			updated[i] = metricFromStored(metric, ms.gauges[metric.ID], 0)
		case model.Counter:
			ms.counters[metric.ID] += *metric.Delta
			updated[i] = metricFromStored(metric, 0, ms.counters[metric.ID])
		}
	}
	return updated, nil
}
