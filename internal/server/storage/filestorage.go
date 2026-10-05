package storage

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"time"

	"go.uber.org/zap"

	"go-musthave-metrics-tpl/internal/model"
)

var _ Storage = (*FileStorage)(nil)

type FileStorage struct {
	mem         *MemStorage
	log         *zap.Logger
	path        string
	synchronous bool
	mu          sync.RWMutex
}

// NewFileStorage создаёт хранилище и при restore читает файл.
// Если чтение не удалось, возвращается рабочее пустое хранилище и ошибка.
func NewFileStorage(log *zap.Logger, path string, synchronous, restore bool) (*FileStorage, error) {
	if log == nil {
		log = zap.NewNop()
	}
	store := &FileStorage{
		mem:         NewMemStorage(),
		log:         log,
		path:        path,
		synchronous: synchronous && path != "",
	}
	if !restore || path == "" {
		return store, nil
	}
	if err := store.Load(); err != nil {
		return store, err
	}
	return store, nil
}

func (s *FileStorage) UpdateGauge(name string, value float64) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.mem.UpdateGauge(name, value)
	s.saveSynced()
}

func (s *FileStorage) UpdateCounter(name string, value int64) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.mem.UpdateCounter(name, value)
	s.saveSynced()
}

func (s *FileStorage) GetGauge(name string) (float64, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.mem.GetGauge(name)
}

func (s *FileStorage) GetCounter(name string) (int64, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.mem.GetCounter(name)
}

func (s *FileStorage) GetAll() (map[string]float64, map[string]int64) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.mem.GetAll()
}

func (s *FileStorage) Load() error {
	if s.path == "" {
		return nil
	}

	data, err := os.ReadFile(s.path)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("read metrics from %s: %w", s.path, err)
	}

	metrics, err := decodeMetrics(data)
	if err != nil {
		return fmt.Errorf("decode metrics from %s: %w", s.path, err)
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	for _, metric := range metrics {
		switch metric.MType {
		case model.Gauge:
			if metric.Value != nil {
				s.mem.UpdateGauge(metric.ID, *metric.Value)
			}
		case model.Counter:
			if metric.Delta != nil {
				s.mem.SetCounter(metric.ID, *metric.Delta)
			}
		}
	}
	return nil
}

func (s *FileStorage) Save() error {
	if s.path == "" {
		return nil
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	return s.writeUnlocked()
}

func (s *FileStorage) FlushEvery(ctx context.Context, interval time.Duration) {
	if s.path == "" || interval <= 0 {
		return
	}

	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := s.Save(); err != nil {
				s.log.Error("failed to save metrics", zap.Error(err))
			}
		}
	}
}

func (s *FileStorage) saveSynced() {
	if !s.synchronous {
		return
	}
	if err := s.writeUnlocked(); err != nil {
		s.log.Error("failed to save metrics", zap.Error(err))
	}
}

func (s *FileStorage) writeUnlocked() error {
	if s.path == "" {
		return nil
	}

	gauges, counters := s.mem.GetAll()
	metrics := make([]model.Metrics, 0, len(gauges)+len(counters))
	for name, value := range gauges {
		v := value
		metrics = append(metrics, model.Metrics{
			ID:    name,
			MType: model.Gauge,
			Value: &v,
		})
	}
	for name, delta := range counters {
		d := delta
		metrics = append(metrics, model.Metrics{
			ID:    name,
			MType: model.Counter,
			Delta: &d,
		})
	}
	sort.Slice(metrics, func(i, j int) bool {
		if metrics[i].ID != metrics[j].ID {
			return metrics[i].ID < metrics[j].ID
		}
		return metrics[i].MType < metrics[j].MType
	})

	data, err := json.MarshalIndent(metrics, "", "  ")
	if err != nil {
		return fmt.Errorf("encode metrics: %w", err)
	}
	data = append(data, '\n')
	if err := writeFileAtomic(s.path, data); err != nil {
		return fmt.Errorf("save metrics to %s: %w", s.path, err)
	}
	return nil
}

func decodeMetrics(data []byte) ([]model.Metrics, error) {
	data = bytes.TrimSpace(data)
	if len(data) == 0 {
		return nil, nil
	}
	if data[0] == '[' {
		var metrics []model.Metrics
		if err := json.Unmarshal(data, &metrics); err != nil {
			return nil, err
		}
		return metrics, nil
	}

	dec := json.NewDecoder(bytes.NewReader(data))
	var metrics []model.Metrics
	for {
		var metric model.Metrics
		if err := dec.Decode(&metric); err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			return nil, err
		}
		metrics = append(metrics, metric)
	}
	return metrics, nil
}

func writeFileAtomic(path string, data []byte) error {
	dir := filepath.Dir(path)
	if dir != "" && dir != "." {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return err
		}
	}

	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0644); err != nil {
		return err
	}

	if err := os.Rename(tmp, path); err == nil {
		return nil
	}

	if err := os.Remove(path); err != nil && !errors.Is(err, os.ErrNotExist) {
		_ = os.Remove(tmp)
		return err
	}
	if err := os.Rename(tmp, path); err != nil {
		_ = os.Remove(tmp)
		return err
	}
	return nil
}
