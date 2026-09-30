package handler

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestValueGaugeAndCounter(t *testing.T) {
	store := newStubStorage()
	store.UpdateGauge("Alloc", 123.45)
	store.UpdateCounter("PollCount", 8)
	handler := NewRouter(store)

	req := httptest.NewRequest(http.MethodGet, "/value/gauge/Alloc", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("gauge status = %d, want %d", rec.Code, http.StatusOK)
	}
	if got := rec.Body.String(); got != "123.45" {
		t.Fatalf("gauge body = %q, want %q", got, "123.45")
	}

	req = httptest.NewRequest(http.MethodGet, "/value/counter/PollCount", nil)
	rec = httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("counter status = %d, want %d", rec.Code, http.StatusOK)
	}
	if got := rec.Body.String(); got != "8" {
		t.Fatalf("counter body = %q, want %q", got, "8")
	}
}

func TestValueUnknownMetric(t *testing.T) {
	handler := NewRouter(newStubStorage())

	tests := []struct {
		name string
		path string
	}{
		{name: "unknown gauge", path: "/value/gauge/Alloc"},
		{name: "unknown counter", path: "/value/counter/PollCount"},
		{name: "unknown type", path: "/value/unknown/Alloc"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, tt.path, nil)
			rec := httptest.NewRecorder()
			handler.ServeHTTP(rec, req)

			if rec.Code != http.StatusNotFound {
				t.Fatalf("status = %d, want %d", rec.Code, http.StatusNotFound)
			}
		})
	}
}

func TestIndexListsMetrics(t *testing.T) {
	store := newStubStorage()
	store.UpdateGauge("Alloc", 42)
	store.UpdateCounter("PollCount", 3)
	handler := NewRouter(store)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	if ct := rec.Header().Get("Content-Type"); !strings.Contains(ct, "text/html") {
		t.Fatalf("Content-Type = %q, want text/html", ct)
	}

	body := rec.Body.String()
	if !strings.Contains(body, "<html") {
		t.Fatalf("body is not HTML: %q", body)
	}
	if !strings.Contains(body, "Alloc") || !strings.Contains(body, "42") {
		t.Fatalf("body missing gauge Alloc: %q", body)
	}
	if !strings.Contains(body, "PollCount") || !strings.Contains(body, "3") {
		t.Fatalf("body missing counter PollCount: %q", body)
	}
}
