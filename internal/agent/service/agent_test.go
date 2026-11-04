package service

import (
	"compress/gzip"
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"syscall"
	"testing"
	"time"

	"github.com/mailru/easyjson"
	"github.com/shirou/gopsutil/v4/cpu"
	"go.uber.org/zap"

	"go-musthave-metrics-tpl/internal/agent/config"
	"go-musthave-metrics-tpl/internal/hash"
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
	var requests []model.MetricsList

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("method = %s, want POST", r.Method)
		}
		if r.URL.Path != "/updates/" {
			t.Errorf("path = %s, want /updates/", r.URL.Path)
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
		var batch model.MetricsList
		if err := easyjson.Unmarshal(body, &batch); err != nil {
			t.Errorf("unmarshal body: %v", err)
			return
		}
		if len(batch) == 0 {
			t.Error("agent sent an empty batch")
		}
		for _, metric := range batch {
			raw, err := easyjson.Marshal(metric)
			if err != nil {
				t.Errorf("marshal metric: %v", err)
				continue
			}
			switch metric.MType {
			case model.Gauge:
				if metric.Value == nil || strings.Contains(string(raw), `"delta"`) {
					t.Errorf("gauge %s body = %s", metric.ID, raw)
				}
			case model.Counter:
				if metric.Delta == nil || strings.Contains(string(raw), `"value"`) {
					t.Errorf("counter %s body = %s", metric.ID, raw)
				}
			default:
				t.Errorf("unexpected metric type %q", metric.MType)
			}
		}
		requests = append(requests, batch)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	s := New(config.DefaultAgent(), zap.NewNop())
	s.serverURL = server.URL
	s.Collect()
	s.Report()

	if len(requests) != 1 {
		t.Fatalf("requests = %d, want 1", len(requests))
	}

	foundPollCount := false
	foundRandomValue := false
	for _, metric := range requests[0] {
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
		got, ok := store.GetGauge(context.Background(), name)
		if !ok || got != value {
			t.Errorf("server gauge %s = (%v, %v), want %v", name, got, ok, value)
		}
	}
	for name, value := range counters {
		got, ok := store.GetCounter(context.Background(), name)
		if !ok || got != value {
			t.Errorf("server counter %s = (%v, %v), want %v", name, got, ok, value)
		}
	}
}

func TestReportSkipsEmptyBatch(t *testing.T) {
	called := false
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	s := New(config.DefaultAgent(), zap.NewNop())
	s.serverURL = server.URL
	s.Report()

	if called {
		t.Fatal("empty batch must not be sent")
	}
}

func TestReportKeepsCountersWhenSendFails(t *testing.T) {
	var requests atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests.Add(1)
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	s := New(config.DefaultAgent(), zap.NewNop())
	s.serverURL = server.URL
	s.Collect()
	s.Report()

	pollCount, ok := s.metrics.Counter("PollCount")
	if !ok || pollCount != 1 {
		t.Fatalf("PollCount = (%d, %v), want (1, true)", pollCount, ok)
	}
	if got := requests.Load(); got != 1 {
		t.Fatalf("requests = %d, want 1", got)
	}
}

func TestReportRetriesConnectionError(t *testing.T) {
	var requests atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests.Add(1)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	next := server.Client().Transport
	var calls atomic.Int32
	s := New(config.DefaultAgent(), zap.NewNop())
	s.serverURL = server.URL
	transport, ok := s.client.Transport.(*retryTransport)
	if !ok {
		t.Fatal("client transport is not retryTransport")
	}
	transport.base = roundTripFunc(func(r *http.Request) (*http.Response, error) {
		if calls.Add(1) == 1 {
			return nil, &net.OpError{Op: "dial", Net: "tcp", Err: syscall.ECONNREFUSED}
		}
		return next.RoundTrip(r)
	})
	s.Collect()

	started := time.Now()
	s.Report()

	if calls.Load() != 2 {
		t.Fatalf("attempts = %d, want 2", calls.Load())
	}
	if requests.Load() != 1 {
		t.Fatalf("accepted requests = %d, want 1", requests.Load())
	}
	if elapsed := time.Since(started); elapsed < 900*time.Millisecond {
		t.Fatalf("retry interval = %s, want at least 1s", elapsed)
	}
	pollCount, ok := s.metrics.Counter("PollCount")
	if !ok || pollCount != 0 {
		t.Fatalf("PollCount = (%d, %v), want (0, true)", pollCount, ok)
	}
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) {
	return f(r)
}

func TestReportKeepsCountersCollectedDuringSend(t *testing.T) {
	started := make(chan struct{})
	release := make(chan struct{})
	var once sync.Once

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		once.Do(func() { close(started) })
		<-release
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	s := New(config.DefaultAgent(), zap.NewNop())
	s.serverURL = server.URL
	s.Collect()

	done := make(chan struct{})
	go func() {
		s.Report()
		close(done)
	}()

	select {
	case <-started:
	case <-time.After(2 * time.Second):
		t.Fatal("report did not reach the server")
	}

	s.Collect()
	close(release)

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("report did not finish")
	}

	pollCount, ok := s.metrics.Counter("PollCount")
	if !ok || pollCount != 1 {
		t.Fatalf("PollCount = (%d, %v), want (1, true)", pollCount, ok)
	}
}

func TestReportSignsCompressedBody(t *testing.T) {
	const key = "secret"
	var gotHash string
	var raw []byte

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotHash = r.Header.Get(hash.Header)
		var err error
		raw, err = io.ReadAll(r.Body)
		if err != nil {
			t.Errorf("read body: %v", err)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	cfg := config.DefaultAgent()
	cfg.Key = key
	s := New(cfg, zap.NewNop())
	s.serverURL = server.URL
	s.Collect()
	s.Report()

	if gotHash == "" {
		t.Fatal("HashSHA256 header was not sent")
	}
	if gotHash != hash.Sum(raw, key) {
		t.Fatalf("HashSHA256 = %q, want HMAC of the compressed body", gotHash)
	}
}

func TestReportWithMatchingKeyIsStored(t *testing.T) {
	const key = "secret"
	store := storage.NewMemStorage()
	server := httptest.NewServer(handler.MetricHandler(nil, store, key))
	defer server.Close()

	cfg := config.DefaultAgent()
	cfg.Key = key
	s := New(cfg, zap.NewNop())
	s.serverURL = server.URL
	s.Collect()
	s.Report()

	pollCount, ok := store.GetCounter(context.Background(), "PollCount")
	if !ok || pollCount != 1 {
		t.Fatalf("server PollCount = (%d, %v), want (1, true)", pollCount, ok)
	}
}

func TestReportWithDifferentKeyIsRejected(t *testing.T) {
	store := storage.NewMemStorage()
	server := httptest.NewServer(handler.MetricHandler(nil, store, "server-key"))
	defer server.Close()

	cfg := config.DefaultAgent()
	cfg.Key = "agent-key"
	s := New(cfg, zap.NewNop())
	s.serverURL = server.URL
	s.Collect()
	s.Report()

	if _, ok := store.GetCounter(context.Background(), "PollCount"); ok {
		t.Fatal("server stored metrics signed with another key")
	}
	pollCount, ok := s.metrics.Counter("PollCount")
	if !ok || pollCount != 1 {
		t.Fatalf("PollCount = (%d, %v), want (1, true)", pollCount, ok)
	}
}

func TestSplitChunks(t *testing.T) {
	metrics := make([]model.Metrics, 5)
	for i := range metrics {
		metrics[i].ID = string(rune('a' + i))
	}

	chunks := splitChunks(metrics, 2)
	if len(chunks) != 2 {
		t.Fatalf("chunks = %d, want 2", len(chunks))
	}
	total := 0
	seen := map[string]int{}
	for _, chunk := range chunks {
		if len(chunk) == 0 {
			t.Fatal("empty chunk")
		}
		total += len(chunk)
		for _, metric := range chunk {
			seen[metric.ID]++
		}
	}
	if total != len(metrics) || len(seen) != len(metrics) {
		t.Fatalf("split lost metrics: total=%d unique=%d", total, len(seen))
	}

	if got := splitChunks(metrics, 1); len(got) != 1 || len(got[0]) != len(metrics) {
		t.Fatalf("limit 1 produced %#v", got)
	}
	if got := splitChunks(metrics, 0); len(got) != 1 {
		t.Fatalf("limit 0 produced %d chunks, want 1", len(got))
	}
	if got := splitChunks(metrics, 100); len(got) != len(metrics) {
		t.Fatalf("limit above length produced %d chunks, want %d", len(got), len(metrics))
	}
	if splitChunks(nil, 3) != nil {
		t.Fatal("empty input must produce no chunks")
	}
}

func TestReportLimitsInFlightRequests(t *testing.T) {
	const limit = 2

	release := make(chan struct{})
	var releaseOnce sync.Once
	unlock := func() { releaseOnce.Do(func() { close(release) }) }
	defer unlock()

	var inFlight atomic.Int32
	var maxInFlight atomic.Int32
	var requests atomic.Int32
	reached := make(chan struct{})
	var once sync.Once

	var mu sync.Mutex
	got := map[string]struct{}{}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		for _, metric := range readBatch(t, r) {
			mu.Lock()
			got[metric.ID] = struct{}{}
			mu.Unlock()
		}

		cur := inFlight.Add(1)
		requests.Add(1)
		for {
			old := maxInFlight.Load()
			if cur <= old || maxInFlight.CompareAndSwap(old, cur) {
				break
			}
		}
		if cur >= int32(limit) {
			once.Do(func() { close(reached) })
		}

		<-release
		inFlight.Add(-1)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	cfg := config.DefaultAgent()
	cfg.RateLimit = limit
	s := New(cfg, zap.NewNop())
	s.serverURL = server.URL
	s.Collect()

	gauges, counters := s.metrics.Snapshot()
	want := len(gauges) + len(counters)

	done := make(chan struct{})
	go func() {
		s.Report()
		close(done)
	}()

	select {
	case <-reached:
	case <-time.After(3 * time.Second):
		t.Fatal("workers did not reach the rate limit")
	}
	if max := maxInFlight.Load(); max > int32(limit) {
		t.Fatalf("in-flight requests = %d, limit %d", max, limit)
	}

	unlock()

	select {
	case <-done:
	case <-time.After(3 * time.Second):
		t.Fatal("report did not finish")
	}
	if max := maxInFlight.Load(); max != int32(limit) {
		t.Fatalf("max in-flight requests = %d, want %d", max, limit)
	}
	if gotReqs := requests.Load(); gotReqs != int32(limit) {
		t.Fatalf("requests = %d, want %d", gotReqs, limit)
	}
	if len(got) != want {
		t.Fatalf("delivered metrics = %d, want %d", len(got), want)
	}
	pollCount, ok := s.metrics.Counter("PollCount")
	if !ok || pollCount != 0 {
		t.Fatalf("PollCount = (%d, %v), want (0, true)", pollCount, ok)
	}
}

func TestCollectSystemAndReport(t *testing.T) {
	var mu sync.Mutex
	got := map[string]model.Metrics{}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		for _, metric := range readBatch(t, r) {
			mu.Lock()
			got[metric.ID] = metric
			mu.Unlock()
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	s := New(config.DefaultAgent(), zap.NewNop())
	s.serverURL = server.URL
	s.collectSystem(context.Background())

	total, ok := s.metrics.Gauge("TotalMemory")
	if !ok || total <= 0 {
		t.Fatalf("TotalMemory = (%v, %v), want a positive gauge", total, ok)
	}
	free, ok := s.metrics.Gauge("FreeMemory")
	if !ok || free < 0 || free > total {
		t.Fatalf("FreeMemory = (%v, %v), TotalMemory = %v", free, ok, total)
	}
	if _, ok := s.metrics.Counter("PollCount"); ok {
		t.Fatal("system collector updated PollCount")
	}

	cpus, err := cpu.Counts(true)
	if err != nil {
		t.Fatalf("cpu.Counts: %v", err)
	}
	for i := 1; i <= cpus; i++ {
		name := fmt.Sprintf("CPUutilization%d", i)
		if _, ok := s.metrics.Gauge(name); !ok {
			t.Fatalf("gauge %s was not collected", name)
		}
	}
	if _, ok := s.metrics.Gauge(fmt.Sprintf("CPUutilization%d", cpus+1)); ok {
		t.Fatalf("unexpected gauge CPUutilization%d", cpus+1)
	}

	s.Report()

	mu.Lock()
	defer mu.Unlock()
	for _, name := range []string{"TotalMemory", "FreeMemory", "CPUutilization1"} {
		metric, ok := got[name]
		if !ok {
			t.Errorf("metric %s was not sent", name)
			continue
		}
		if metric.MType != model.Gauge || metric.Value == nil {
			t.Errorf("%s = %+v, want gauge", name, metric)
		}
	}
}

func TestRunStopsOnCancel(t *testing.T) {
	s := New(config.DefaultAgent(), zap.NewNop())
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	done := make(chan struct{})
	go func() {
		s.Run(ctx)
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("Run did not stop after cancel")
	}
}

func readBatch(t *testing.T, r *http.Request) []model.Metrics {
	t.Helper()
	if r.URL.Path != "/updates/" {
		t.Errorf("path = %s, want /updates/", r.URL.Path)
	}
	zr, err := gzip.NewReader(r.Body)
	if err != nil {
		t.Errorf("gzip reader: %v", err)
		return nil
	}
	defer zr.Close()
	body, err := io.ReadAll(zr)
	if err != nil {
		t.Errorf("read body: %v", err)
		return nil
	}
	var batch model.MetricsList
	if err := easyjson.Unmarshal(body, &batch); err != nil {
		t.Errorf("unmarshal body: %v", err)
		return nil
	}
	return batch
}
