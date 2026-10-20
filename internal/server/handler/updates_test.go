package handler

import (
	"bytes"
	"compress/gzip"
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/mailru/easyjson"

	"go-musthave-metrics-tpl/internal/model"
	"go-musthave-metrics-tpl/internal/server/storage"
)

func TestUpdatesBatch(t *testing.T) {
	store := newStubStorage()
	h := NewRouter(nil, store)

	body := `[
		{"id":"Alloc","type":"gauge","value":1.5},
		{"id":"PollCount","type":"counter","delta":2},
		{"id":"PollCount","type":"counter","delta":3},
		{"id":"Zero","type":"gauge","value":0}
	]`
	req := httptest.NewRequest(http.MethodPost, "/updates/", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d, body = %s", rec.Code, http.StatusOK, rec.Body.String())
	}
	if ct := rec.Header().Get("Content-Type"); ct != "application/json" {
		t.Fatalf("Content-Type = %q, want application/json", ct)
	}
	if store.gauges["Alloc"] != 1.5 {
		t.Fatalf("Alloc = %v, want 1.5", store.gauges["Alloc"])
	}
	if store.counters["PollCount"] != 5 {
		t.Fatalf("PollCount = %d, want 5", store.counters["PollCount"])
	}
	if store.gauges["Zero"] != 0 {
		t.Fatalf("Zero = %v, want 0", store.gauges["Zero"])
	}

	var got model.MetricsList
	if err := easyjson.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if len(got) != 4 {
		t.Fatalf("response len = %d, want 4", len(got))
	}
	if got[0].Value == nil || *got[0].Value != 1.5 || got[0].Delta != nil {
		t.Fatalf("gauge response = %+v", got[0])
	}
	if strings.Contains(string(mustMarshalMetric(t, got[0])), `"delta"`) {
		t.Fatalf("gauge response must omit delta: %s", mustMarshalMetric(t, got[0]))
	}
	if got[1].Delta == nil || *got[1].Delta != 2 {
		t.Fatalf("first counter response = %+v, want delta 2", got[1])
	}
	if got[2].Delta == nil || *got[2].Delta != 5 {
		t.Fatalf("second counter response = %+v, want delta 5", got[2])
	}
	if got[3].Value == nil || *got[3].Value != 0 || !strings.Contains(string(mustMarshalMetric(t, got[3])), `"value":0`) {
		t.Fatalf("zero gauge response = %s", mustMarshalMetric(t, got[3]))
	}

	req = httptest.NewRequest(http.MethodGet, "/value/counter/PollCount", nil)
	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK || rec.Body.String() != "5" {
		t.Fatalf("text value = %d %q, want 200 5", rec.Code, rec.Body.String())
	}
}

func TestUpdatesWithoutSlash(t *testing.T) {
	store := storage.NewMemStorage()
	h := NewRouter(nil, store)

	req := httptest.NewRequest(http.MethodPost, "/updates", strings.NewReader(
		`[{"id":"Alloc","type":"gauge","value":4}]`,
	))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	got, ok := store.GetGauge(context.Background(), "Alloc")
	if !ok || got != 4 {
		t.Fatalf("Alloc = (%v, %v), want (4, true)", got, ok)
	}
}

func TestUpdatesEmptyBatch(t *testing.T) {
	store := newStubStorage()
	store.UpdateCounter(context.Background(), "PollCount", 1)
	h := NewRouter(nil, store)

	req := httptest.NewRequest(http.MethodPost, "/updates/", strings.NewReader(`[]`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	if strings.TrimSpace(rec.Body.String()) != "[]" {
		t.Fatalf("body = %s, want []", rec.Body.String())
	}
	if store.counters["PollCount"] != 1 {
		t.Fatalf("PollCount = %d, want 1", store.counters["PollCount"])
	}
}

func TestUpdatesInvalidRequests(t *testing.T) {
	tests := []struct {
		name string
		body string
		want int
	}{
		{name: "invalid json", body: `{`, want: http.StatusBadRequest},
		{name: "single object", body: `{"id":"Alloc","type":"gauge","value":1}`, want: http.StatusBadRequest},
		{name: "null", body: `null`, want: http.StatusBadRequest},
		{name: "unknown type", body: `[{"id":"Alloc","type":"unknown","value":1}]`, want: http.StatusBadRequest},
		{name: "gauge without value", body: `[{"id":"Alloc","type":"gauge"}]`, want: http.StatusBadRequest},
		{name: "counter without delta", body: `[{"id":"PollCount","type":"counter"}]`, want: http.StatusBadRequest},
		{name: "empty id", body: `[{"id":"","type":"gauge","value":1}]`, want: http.StatusNotFound},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := newStubStorage()
			store.UpdateCounter(context.Background(), "PollCount", 7)
			h := NewRouter(nil, store)

			req := httptest.NewRequest(http.MethodPost, "/updates/", strings.NewReader(tt.body))
			req.Header.Set("Content-Type", "application/json")
			rec := httptest.NewRecorder()
			h.ServeHTTP(rec, req)

			if rec.Code != tt.want {
				t.Fatalf("status = %d, want %d", rec.Code, tt.want)
			}
			if _, ok := store.gauges["Alloc"]; ok {
				t.Fatal("invalid batch must not store a gauge")
			}
			if store.counters["PollCount"] != 7 {
				t.Fatalf("PollCount = %d, want 7", store.counters["PollCount"])
			}
		})
	}
}

func TestUpdatesDoesNotApplyPartialBatch(t *testing.T) {
	store := newStubStorage()
	h := NewRouter(nil, store)

	req := httptest.NewRequest(http.MethodPost, "/updates/", strings.NewReader(
		`[{"id":"Alloc","type":"gauge","value":1},{"id":"Bad","type":"unknown","value":1}]`,
	))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
	if _, ok := store.gauges["Alloc"]; ok {
		t.Fatal("Alloc was stored from an invalid batch")
	}
}

func TestUpdatesGzip(t *testing.T) {
	store := newStubStorage()
	h := NewRouter(nil, store)

	var buf bytes.Buffer
	zw := gzip.NewWriter(&buf)
	_, _ = zw.Write([]byte(`[{"id":"Alloc","type":"gauge","value":8.5},{"id":"PollCount","type":"counter","delta":4}]`))
	_ = zw.Close()

	req := httptest.NewRequest(http.MethodPost, "/updates/", &buf)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Content-Encoding", "gzip")
	req.Header.Set("Accept-Encoding", "gzip")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	if rec.Header().Get("Content-Encoding") != "gzip" {
		t.Fatalf("Content-Encoding = %q, want gzip", rec.Header().Get("Content-Encoding"))
	}
	if store.gauges["Alloc"] != 8.5 || store.counters["PollCount"] != 4 {
		t.Fatalf("stored gauge=%v counter=%d", store.gauges["Alloc"], store.counters["PollCount"])
	}

	zr, err := gzip.NewReader(bytes.NewReader(rec.Body.Bytes()))
	if err != nil {
		t.Fatalf("gzip reader: %v", err)
	}
	payload, err := io.ReadAll(zr)
	_ = zr.Close()
	if err != nil {
		t.Fatalf("read gzip: %v", err)
	}
	var got model.MetricsList
	if err := easyjson.Unmarshal(payload, &got); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if len(got) != 2 || got[1].Delta == nil || *got[1].Delta != 4 {
		t.Fatalf("response = %+v", got)
	}
}

func mustMarshalMetric(t *testing.T, metric model.Metrics) []byte {
	t.Helper()
	payload, err := easyjson.Marshal(metric)
	if err != nil {
		t.Fatalf("marshal metric: %v", err)
	}
	return payload
}
