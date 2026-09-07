package handler

import (
	"go-musthave-metrics-tpl/internal/model/server"
	"net/http"
	"net/http/httptest"
	"testing"
)

type stubStorage struct {
	gauges   map[string]float64
	counters map[string]int64
}

func newStubStorage() *stubStorage {
	return &stubStorage{
		gauges:   make(map[string]float64),
		counters: make(map[string]int64),
	}
}

func (s *stubStorage) UpdateGauge(name string, value float64) {
	s.gauges[name] = value
}

func (s *stubStorage) UpdateCounter(name string, value int64) {
	s.counters[name] += value
}

func TestUpdateGauge(t *testing.T) {
	store := newStubStorage()
	handler := Update(store)

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
	handler := Update(store)

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
	store := server.NewMemStorage()
	handler := Update(store)

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
	h := MetricHandler()
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
