package agent

import (
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
)

func TestCollectUpdatesRuntimeMetrics(t *testing.T) {
	s := New()
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
		if _, ok := s.GetMetrics().Gauge(name); !ok {
			t.Errorf("gauge %q was not collected", name)
		}
	}

	pollCount, ok := s.GetMetrics().Counter("PollCount")
	if !ok {
		t.Fatal("counter PollCount was not collected")
	}
	if pollCount != 1 {
		t.Fatalf("PollCount = %d, want 1", pollCount)
	}

	s.Collect()
	pollCount, _ = s.GetMetrics().Counter("PollCount")
	if pollCount != 2 {
		t.Fatalf("PollCount after second collect = %d, want 2", pollCount)
	}
}

func TestReportSendsMetrics(t *testing.T) {
	var requests []string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("method = %s, want POST", r.Method)
		}
		if ct := r.Header.Get("Content-Type"); ct != "text/plain" {
			t.Errorf("Content-Type = %q, want text/plain", ct)
		}
		requests = append(requests, r.URL.Path)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	s := New()
	s.SetServerURL(server.URL)
	s.Collect()
	s.Report()

	if len(requests) == 0 {
		t.Fatal("expected at least one request to the server")
	}

	foundPollCount := false
	foundRandomValue := false
	for _, path := range requests {
		if strings.HasPrefix(path, "/update/counter/PollCount/") {
			foundPollCount = true
			value := strings.TrimPrefix(path, "/update/counter/PollCount/")
			if _, err := strconv.ParseInt(value, 10, 64); err != nil {
				t.Errorf("invalid PollCount value %q: %v", value, err)
			}
		}
		if strings.HasPrefix(path, "/update/gauge/RandomValue/") {
			foundRandomValue = true
		}
	}

	if !foundPollCount {
		t.Error("PollCount was not sent")
	}
	if !foundRandomValue {
		t.Error("RandomValue was not sent")
	}

	pollCount, ok := s.GetMetrics().Counter("PollCount")
	if !ok {
		t.Fatal("PollCount missing after report")
	}
	if pollCount != 0 {
		t.Fatalf("PollCount after successful report = %d, want 0", pollCount)
	}
}
