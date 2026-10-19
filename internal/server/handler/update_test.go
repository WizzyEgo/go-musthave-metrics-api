package handler

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest/observer"

	"go-musthave-metrics-tpl/internal/model"
	"go-musthave-metrics-tpl/internal/server/storage"
)

type stubStorage struct {
	gauges   map[string]float64
	counters map[string]int64
	pingErr  error
}

func newStubStorage() *stubStorage {
	return &stubStorage{
		gauges:   make(map[string]float64),
		counters: make(map[string]int64),
	}
}

func (s *stubStorage) UpdateGauge(ctx context.Context, name string, value float64) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	s.gauges[name] = value
	return nil
}

func (s *stubStorage) UpdateCounter(ctx context.Context, name string, value int64) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	s.counters[name] += value
	return nil
}

func (s *stubStorage) GetGauge(ctx context.Context, name string) (float64, bool) {
	if err := ctx.Err(); err != nil {
		return 0, false
	}
	value, ok := s.gauges[name]
	return value, ok
}

func (s *stubStorage) GetCounter(ctx context.Context, name string) (int64, bool) {
	if err := ctx.Err(); err != nil {
		return 0, false
	}
	value, ok := s.counters[name]
	return value, ok
}

func (s *stubStorage) UpdateBatch(ctx context.Context, metrics []model.Metrics) ([]model.Metrics, error) {
	for _, metric := range metrics {
		if metric.ID == "" {
			return nil, storage.ErrEmptyMetricID
		}
		switch metric.MType {
		case model.Gauge:
			if metric.Value == nil {
				return nil, storage.ErrInvalidMetric
			}
		case model.Counter:
			if metric.Delta == nil {
				return nil, storage.ErrInvalidMetric
			}
		default:
			return nil, storage.ErrInvalidMetric
		}
	}

	updated := make([]model.Metrics, len(metrics))
	for i, metric := range metrics {
		switch metric.MType {
		case model.Gauge:
			if err := s.UpdateGauge(ctx, metric.ID, *metric.Value); err != nil {
				return nil, err
			}
			value := s.gauges[metric.ID]
			updated[i] = model.Metrics{ID: metric.ID, MType: model.Gauge, Value: &value, Hash: metric.Hash}
		case model.Counter:
			if err := s.UpdateCounter(ctx, metric.ID, *metric.Delta); err != nil {
				return nil, err
			}
			delta := s.counters[metric.ID]
			updated[i] = model.Metrics{ID: metric.ID, MType: model.Counter, Delta: &delta, Hash: metric.Hash}
		}
	}
	return updated, nil
}

func (s *stubStorage) GetAll(ctx context.Context) (map[string]float64, map[string]int64) {
	if err := ctx.Err(); err != nil {
		return map[string]float64{}, map[string]int64{}
	}
	gauges := make(map[string]float64, len(s.gauges))
	for name, value := range s.gauges {
		gauges[name] = value
	}
	counters := make(map[string]int64, len(s.counters))
	for name, value := range s.counters {
		counters[name] = value
	}
	return gauges, counters
}

func (s *stubStorage) Ping(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	return s.pingErr
}

func TestUpdateGauge(t *testing.T) {
	store := newStubStorage()
	handler := NewRouter(nil, store)

	req := httptest.NewRequest(http.MethodPost, "/update/gauge/Alloc/123.45", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	if got := store.gauges["Alloc"]; got != 123.45 {
		t.Fatalf("Alloc = %v, want 123.45", got)
	}
}

func TestUpdateCounter(t *testing.T) {
	store := newStubStorage()
	handler := NewRouter(nil, store)

	req := httptest.NewRequest(http.MethodPost, "/update/counter/PollCount/5", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	if got := store.counters["PollCount"]; got != 5 {
		t.Fatalf("PollCount = %d, want 5", got)
	}

	req = httptest.NewRequest(http.MethodPost, "/update/counter/PollCount/3", nil)
	rec = httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if got := store.counters["PollCount"]; got != 8 {
		t.Fatalf("PollCount after second update = %d, want 8", got)
	}
}

func TestUpdateInvalidRequests(t *testing.T) {
	store := storage.NewMemStorage()
	handler := NewRouter(nil, store)

	tests := []struct {
		name   string
		method string
		path   string
		want   int
	}{
		{name: "method not allowed", method: http.MethodGet, path: "/update/gauge/Alloc/1", want: http.StatusMethodNotAllowed},
		{name: "unknown type", method: http.MethodPost, path: "/update/unknown/Alloc/1", want: http.StatusBadRequest},
		{name: "invalid gauge", method: http.MethodPost, path: "/update/gauge/Alloc/abc", want: http.StatusBadRequest},
		{name: "invalid counter", method: http.MethodPost, path: "/update/counter/PollCount/1.5", want: http.StatusBadRequest},
		{name: "missing name", method: http.MethodPost, path: "/update/gauge/", want: http.StatusNotFound},
		{name: "short path", method: http.MethodPost, path: "/update/gauge/Alloc", want: http.StatusNotFound},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, tt.path, nil)
			rec := httptest.NewRecorder()
			handler.ServeHTTP(rec, req)

			if rec.Code != tt.want {
				t.Fatalf("status = %d, want %d", rec.Code, tt.want)
			}
		})
	}
}

func TestMetricHandler(t *testing.T) {
	h := MetricHandler(nil, storage.NewMemStorage())
	req := httptest.NewRequest(http.MethodPost, "/update/gauge/HeapAlloc/42", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	if ct := rec.Header().Get("Content-Type"); ct != "text/plain" {
		t.Fatalf("Content-Type = %q, want text/plain", ct)
	}
}

func TestRouterLogsRequests(t *testing.T) {
	core, recorded := observer.New(zapcore.InfoLevel)
	store := newStubStorage()
	store.UpdateGauge(context.Background(), "SingletonAlloc", 1)
	h := NewRouter(zap.New(core), store)

	tests := []struct {
		method string
		path   string
		status int
	}{
		{method: http.MethodPost, path: "/update/counter/SingletonPoll/1", status: http.StatusOK},
		{method: http.MethodGet, path: "/value/gauge/SingletonAlloc", status: http.StatusOK},
		{method: http.MethodGet, path: "/", status: http.StatusOK},
	}

	for _, tt := range tests {
		t.Run(tt.method+" "+tt.path, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, tt.path, nil)
			rec := httptest.NewRecorder()
			h.ServeHTTP(rec, req)

			if rec.Code != tt.status {
				t.Fatalf("status = %d, want %d", rec.Code, tt.status)
			}

			fields := logFields(recorded.All(), tt.path)
			if fields == nil {
				t.Fatalf("no log entry for %s", tt.path)
			}
			if fields["method"] != tt.method {
				t.Fatalf("method = %v, want %s", fields["method"], tt.method)
			}
			duration, ok := fields["duration"].(time.Duration)
			if !ok || duration < 0 {
				t.Fatalf("duration = %v (%T), want non-negative time.Duration", fields["duration"], fields["duration"])
			}
			if fields["status"] != int64(tt.status) {
				t.Fatalf("status = %v, want %d", fields["status"], tt.status)
			}
			size, ok := fields["size"].(int64)
			if !ok || size < 0 {
				t.Fatalf("size = %v (%T), want non-negative int64", fields["size"], fields["size"])
			}
		})
	}
}

func logFields(entries []observer.LoggedEntry, uri string) map[string]any {
	for i := len(entries) - 1; i >= 0; i-- {
		fields := entries[i].ContextMap()
		if fields["uri"] == uri {
			return fields
		}
	}
	return nil
}
