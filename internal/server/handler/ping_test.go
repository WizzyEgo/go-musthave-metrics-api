package handler

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestPingOK(t *testing.T) {
	h := NewRouter(nil, newStubStorage())
	req := httptest.NewRequest(http.MethodGet, "/ping", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
}

func TestPingUnavailable(t *testing.T) {
	store := newStubStorage()
	store.pingErr = errors.New("down")
	h := NewRouter(nil, store)
	req := httptest.NewRequest(http.MethodGet, "/ping", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusInternalServerError)
	}
	if body := strings.TrimSpace(rec.Body.String()); body != "" {
		t.Fatalf("body = %q, want empty", body)
	}
}

func TestPingInMemoryStorage(t *testing.T) {
	h := NewRouter(nil, newStubStorage())
	req := httptest.NewRequest(http.MethodGet, "/ping", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
}
