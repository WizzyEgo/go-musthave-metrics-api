package handler

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/mailru/easyjson"

	"go-musthave-metrics-tpl/internal/server/model"
)

func TestUpdateJSONGauge(t *testing.T) {
	store := newStubStorage()
	h := NewRouter(store)

	req := httptest.NewRequest(http.MethodPost, "/update", strings.NewReader(
		`{"id":"LastGC","type":"gauge","value":1744184459}`,
	))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	if ct := rec.Header().Get("Content-Type"); ct != "application/json" {
		t.Fatalf("Content-Type = %q, want application/json", ct)
	}

	var got model.Metrics
	if err := easyjson.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if got.ID != "LastGC" || got.MType != model.Gauge || got.Value == nil || *got.Value != 1744184459 {
		t.Fatalf("response = %+v", got)
	}
	if got.Delta != nil {
		t.Fatalf("delta = %v, want nil", *got.Delta)
	}
	if strings.Contains(rec.Body.String(), `"delta"`) {
		t.Fatalf("gauge response must omit delta: %s", rec.Body.String())
	}
	if store.gauges["LastGC"] != 1744184459 {
		t.Fatalf("stored LastGC = %v", store.gauges["LastGC"])
	}
}

func TestUpdateJSONGaugeZero(t *testing.T) {
	store := newStubStorage()
	h := NewRouter(store)

	req := httptest.NewRequest(http.MethodPost, "/update/", strings.NewReader(
		`{"id":"Alloc","type":"gauge","value":0}`,
	))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	if store.gauges["Alloc"] != 0 {
		t.Fatalf("stored Alloc = %v, want 0", store.gauges["Alloc"])
	}
	if !strings.Contains(rec.Body.String(), `"value":0`) {
		t.Fatalf("response = %s, want value 0", rec.Body.String())
	}
}

func TestUpdateJSONCounterAccumulates(t *testing.T) {
	store := newStubStorage()
	h := NewRouter(store)

	send := func(body string) *httptest.ResponseRecorder {
		req := httptest.NewRequest(http.MethodPost, "/update", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)
		return rec
	}

	rec := send(`{"id":"PollCount","type":"counter","delta":5}`)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	if ct := rec.Header().Get("Content-Type"); ct != "application/json" {
		t.Fatalf("Content-Type = %q, want application/json", ct)
	}

	rec = send(`{"id":"PollCount","type":"counter","delta":3}`)
	if rec.Code != http.StatusOK {
		t.Fatalf("second status = %d, want %d", rec.Code, http.StatusOK)
	}

	var got model.Metrics
	if err := easyjson.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if got.ID != "PollCount" || got.MType != model.Counter || got.Delta == nil || *got.Delta != 8 {
		t.Fatalf("response = %+v, want delta 8", got)
	}
	if got.Value != nil || strings.Contains(rec.Body.String(), `"value"`) {
		t.Fatalf("counter response must omit value: %s", rec.Body.String())
	}
	if store.counters["PollCount"] != 8 {
		t.Fatalf("stored PollCount = %d, want 8", store.counters["PollCount"])
	}
}

func TestUpdateJSONInvalidRequests(t *testing.T) {
	h := NewRouter(newStubStorage())

	tests := []struct {
		name string
		body string
		want int
	}{
		{name: "invalid json", body: `{`, want: http.StatusBadRequest},
		{name: "unknown type", body: `{"id":"Alloc","type":"unknown","value":1}`, want: http.StatusBadRequest},
		{name: "gauge without value", body: `{"id":"Alloc","type":"gauge"}`, want: http.StatusBadRequest},
		{name: "counter without delta", body: `{"id":"PollCount","type":"counter"}`, want: http.StatusBadRequest},
		{name: "empty id", body: `{"id":"","type":"gauge","value":1}`, want: http.StatusNotFound},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/update", strings.NewReader(tt.body))
			req.Header.Set("Content-Type", "application/json")
			rec := httptest.NewRecorder()
			h.ServeHTTP(rec, req)

			if rec.Code != tt.want {
				t.Fatalf("status = %d, want %d", rec.Code, tt.want)
			}
		})
	}
}

func TestValueJSON(t *testing.T) {
	store := newStubStorage()
	store.UpdateGauge("LastGC", 1744184459)
	store.UpdateCounter("PollCount", 8)
	h := NewRouter(store)

	req := httptest.NewRequest(http.MethodPost, "/value", strings.NewReader(
		`{"id":"LastGC","type":"gauge"}`,
	))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("gauge status = %d, want %d", rec.Code, http.StatusOK)
	}
	if ct := rec.Header().Get("Content-Type"); ct != "application/json" {
		t.Fatalf("Content-Type = %q, want application/json", ct)
	}

	var gauge model.Metrics
	if err := easyjson.Unmarshal(rec.Body.Bytes(), &gauge); err != nil {
		t.Fatalf("unmarshal gauge: %v", err)
	}
	if gauge.ID != "LastGC" || gauge.MType != model.Gauge || gauge.Value == nil || *gauge.Value != 1744184459 || gauge.Delta != nil {
		t.Fatalf("gauge response = %+v", gauge)
	}

	req = httptest.NewRequest(http.MethodPost, "/value/", strings.NewReader(
		`{"id":"PollCount","type":"counter"}`,
	))
	req.Header.Set("Content-Type", "application/json")
	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("counter status = %d, want %d", rec.Code, http.StatusOK)
	}
	var counter model.Metrics
	if err := easyjson.Unmarshal(rec.Body.Bytes(), &counter); err != nil {
		t.Fatalf("unmarshal counter: %v", err)
	}
	if counter.ID != "PollCount" || counter.Delta == nil || *counter.Delta != 8 || counter.Value != nil {
		t.Fatalf("counter response = %+v", counter)
	}
}

func TestValueJSONErrors(t *testing.T) {
	store := newStubStorage()
	store.UpdateGauge("Alloc", 1)
	h := NewRouter(store)

	tests := []struct {
		name string
		body string
		want int
	}{
		{name: "invalid json", body: `{`, want: http.StatusBadRequest},
		{name: "unknown gauge", body: `{"id":"Missing","type":"gauge"}`, want: http.StatusNotFound},
		{name: "unknown counter", body: `{"id":"Missing","type":"counter"}`, want: http.StatusNotFound},
		{name: "unknown type", body: `{"id":"Alloc","type":"unknown"}`, want: http.StatusNotFound},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/value", strings.NewReader(tt.body))
			req.Header.Set("Content-Type", "application/json")
			rec := httptest.NewRecorder()
			h.ServeHTTP(rec, req)

			if rec.Code != tt.want {
				t.Fatalf("status = %d, want %d", rec.Code, tt.want)
			}
		})
	}
}

func TestJSONUpdateIsReadableByTextAPI(t *testing.T) {
	h := NewRouter(newStubStorage())

	req := httptest.NewRequest(http.MethodPost, "/update", strings.NewReader(
		`{"id":"Alloc","type":"gauge","value":42.5}`,
	))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("update status = %d, want %d", rec.Code, http.StatusOK)
	}

	req = httptest.NewRequest(http.MethodGet, "/value/gauge/Alloc", nil)
	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("value status = %d, want %d", rec.Code, http.StatusOK)
	}
	if rec.Body.String() != "42.5" {
		t.Fatalf("body = %q, want 42.5", rec.Body.String())
	}
}
