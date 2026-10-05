package service

import (
	"compress/gzip"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/mailru/easyjson"
	"go.uber.org/zap"

	"go-musthave-metrics-tpl/internal/agent/config"
	"go-musthave-metrics-tpl/internal/model"
	"go-musthave-metrics-tpl/internal/server/handler"
	"go-musthave-metrics-tpl/internal/server/storage"
)

func TestCollectReturnsRuntimeGauges(t *testing.T) {
	s := New(config.DefaultAgent(), zap.NewNop())
	gauges := s.collect()

	required := []string{
		"Alloc", "BuckHashSys", "Frees", "GCCPUFraction", "GCSys",
		"HeapAlloc", "HeapIdle", "HeapInuse", "HeapObjects", "HeapReleased",
		"HeapSys", "LastGC", "Lookups", "MCacheInuse", "MCacheSys",
		"MSpanInuse", "MSpanSys", "Mallocs", "NextGC", "NumForcedGC",
		"NumGC", "OtherSys", "PauseTotalNs", "StackInuse", "StackSys",
		"Sys", "TotalAlloc", "RandomValue",
	}

	if len(gauges) != len(required) {
		t.Fatalf("collect() returned %d gauges, want %d", len(gauges), len(required))
	}
	for _, name := range required {
		if _, ok := gauges[name]; !ok {
			t.Errorf("collect() missing gauge %q", name)
		}
	}
}

func TestCollectUpdatesRuntimeMetrics(t *testing.T) {
	s := New(config.DefaultAgent(), zap.NewNop())
	s.Collect()

	requiredGauges := []string{
		"Alloc", "BuckHashSys", "Frees", "GCCPUFraction", "GCSys",
		"HeapAlloc", "HeapIdle", "HeapInuse", "HeapObjects", "HeapReleased",
		"HeapSys", "LastGC", "Lookups", "MCacheInuse", "MCacheSys",
		"MSpanInuse", "MSpanSys", "Mallocs", "NextGC", "NumForcedGC",
		"NumGC", "OtherSys", "PauseTotalNs", "StackInuse", "StackSys",
		"Sys", "TotalAlloc", "RandomValue",
	}

	for _, name := range requiredGauges {
		if _, ok := s.metrics.Gauge(name); !ok {
			t.Errorf("gauge %q was not collected", name)
		}
	}

	pollCount, ok := s.metrics.Counter("PollCount")
	if !ok {
		t.Fatal("counter PollCount was not collected")
	}
	if pollCount != 1 {
		t.Fatalf("PollCount = %d, want 1", pollCount)
	}

	s.Collect()
	pollCount, _ = s.metrics.Counter("PollCount")
	if pollCount != 2 {
		t.Fatalf("PollCount after second collect = %d, want 2", pollCount)
	}
}

func TestReportSendsMetrics(t *testing.T) {
	var requests []model.Metrics

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("method = %s, want POST", r.Method)
		}
		if r.URL.Path != "/update" {
			t.Errorf("path = %s, want /update", r.URL.Path)
		}
		if ct := r.Header.Get("Content-Type"); ct != "application/json" {
			t.Errorf("Content-Type = %q, want application/json", ct)
		}
		if ce := r.Header.Get("Content-Encoding"); ce != "gzip" {
			t.Errorf("Content-Encoding = %q, want gzip", ce)
		}

		zr, err := gzip.NewReader(r.Body)
		if err != nil {
			t.Errorf("gzip reader: %v", err)
			return
		}
		body, err := io.ReadAll(zr)
		if err != nil {
			t.Errorf("read body: %v", err)
		}
		_ = zr.Close()
		var metric model.Metrics
		if err := easyjson.Unmarshal(body, &metric); err != nil {
			t.Errorf("unmarshal body: %v", err)
		}
		raw := string(body)
		switch metric.MType {
		case model.Gauge:
			if metric.Value == nil || strings.Contains(raw, `"delta"`) {
				t.Errorf("gauge %s body = %s", metric.ID, raw)
			}
		case model.Counter:
			if metric.Delta == nil || strings.Contains(raw, `"value"`) {
				t.Errorf("counter %s body = %s", metric.ID, raw)
			}
		default:
			t.Errorf("unexpected metric type %q", metric.MType)
		}
		requests = append(requests, metric)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	s := New(config.DefaultAgent(), zap.NewNop())
	s.serverURL = server.URL
	s.Collect()
	s.Report()

	if len(requests) == 0 {
		t.Fatal("expected at least one request to the server")
	}

	foundPollCount := false
	foundRandomValue := false
	for _, metric := range requests {
		if metric.MType == model.Counter && metric.ID == "PollCount" {
			foundPollCount = true
			if metric.Delta == nil || *metric.Delta != 1 {
				t.Errorf("PollCount delta = %v, want 1", metric.Delta)
			}
		}
		if metric.MType == model.Gauge && metric.ID == "RandomValue" {
			foundRandomValue = true
		}
	}

	if !foundPollCount {
		t.Error("PollCount was not sent")
	}
	if !foundRandomValue {
		t.Error("RandomValue was not sent")
	}

	pollCount, ok := s.metrics.Counter("PollCount")
	if !ok {
		t.Fatal("PollCount missing after report")
	}
	if pollCount != 0 {
		t.Fatalf("PollCount after successful report = %d, want 0", pollCount)
	}
}

func TestReportStoresMetricsOnServer(t *testing.T) {
	store := storage.NewMemStorage()
	server := httptest.NewServer(handler.NewRouter(nil, store))
	defer server.Close()

	s := New(config.DefaultAgent(), zap.NewNop())
	s.serverURL = server.URL
	s.Collect()

	gauges, counters := s.metrics.Snapshot()
	s.Report()

	for name, value := range gauges {
		got, ok := store.GetGauge(name)
		if !ok || got != value {
			t.Errorf("server gauge %s = (%v, %v), want %v", name, got, ok, value)
		}
	}
	for name, value := range counters {
		got, ok := store.GetCounter(name)
		if !ok || got != value {
			t.Errorf("server counter %s = (%v, %v), want %v", name, got, ok, value)
		}
	}
}
