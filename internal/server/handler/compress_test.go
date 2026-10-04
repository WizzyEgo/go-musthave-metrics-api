package handler

import (
	"bytes"
	"compress/gzip"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/mailru/easyjson"

	"go-musthave-metrics-tpl/internal/model"
)

func TestRouterGzipJSONAndHTML(t *testing.T) {
	store := newStubStorage()
	store.UpdateGauge("Alloc", 42)
	h := NewRouter(nil, store)

	var buf bytes.Buffer
	zw := gzip.NewWriter(&buf)
	_, _ = zw.Write([]byte(`{"id":"LastGC","type":"gauge","value":1744184459}`))
	_ = zw.Close()

	req := httptest.NewRequest(http.MethodPost, "/update", &buf)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Content-Encoding", "gzip")
	req.Header.Set("Accept-Encoding", "gzip")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("update status = %d, want %d", rec.Code, http.StatusOK)
	}
	if rec.Header().Get("Content-Encoding") != "gzip" {
		t.Fatalf("Content-Encoding = %q, want gzip", rec.Header().Get("Content-Encoding"))
	}
	if rec.Header().Get("Content-Type") != "application/json" {
		t.Fatalf("Content-Type = %q, want application/json", rec.Header().Get("Content-Type"))
	}

	var updated model.Metrics
	if err := easyjson.Unmarshal(gunzipBody(t, rec.Body.Bytes()), &updated); err != nil {
		t.Fatalf("unmarshal update response: %v", err)
	}
	if updated.Value == nil || *updated.Value != 1744184459 {
		t.Fatalf("update response = %+v", updated)
	}

	req = httptest.NewRequest(http.MethodPost, "/value", strings.NewReader(`{"id":"LastGC","type":"gauge"}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept-Encoding", "gzip")
	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("value status = %d, want %d", rec.Code, http.StatusOK)
	}
	if rec.Header().Get("Content-Encoding") != "gzip" {
		t.Fatalf("value Content-Encoding = %q, want gzip", rec.Header().Get("Content-Encoding"))
	}
	var got model.Metrics
	if err := easyjson.Unmarshal(gunzipBody(t, rec.Body.Bytes()), &got); err != nil {
		t.Fatalf("unmarshal value response: %v", err)
	}
	if got.Value == nil || *got.Value != 1744184459 {
		t.Fatalf("value response = %+v", got)
	}

	req = httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Accept-Encoding", "gzip")
	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("index status = %d, want %d", rec.Code, http.StatusOK)
	}
	if rec.Header().Get("Content-Encoding") != "gzip" {
		t.Fatalf("index Content-Encoding = %q, want gzip", rec.Header().Get("Content-Encoding"))
	}
	if ct := rec.Header().Get("Content-Type"); !strings.Contains(ct, "text/html") {
		t.Fatalf("index Content-Type = %q, want text/html", ct)
	}
	page := string(gunzipBody(t, rec.Body.Bytes()))
	if !strings.Contains(page, "<html") || !strings.Contains(page, "Alloc") {
		t.Fatalf("index body = %q", page)
	}
}

func TestRouterDoesNotGzipPlainText(t *testing.T) {
	store := newStubStorage()
	store.UpdateGauge("Alloc", 42.5)
	h := NewRouter(nil, store)

	req := httptest.NewRequest(http.MethodGet, "/value/gauge/Alloc", nil)
	req.Header.Set("Accept-Encoding", "gzip")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	if rec.Header().Get("Content-Encoding") != "" {
		t.Fatalf("Content-Encoding = %q, want empty", rec.Header().Get("Content-Encoding"))
	}
	if rec.Body.String() != "42.5" {
		t.Fatalf("body = %q, want 42.5", rec.Body.String())
	}
}

func gunzipBody(t *testing.T, data []byte) []byte {
	t.Helper()
	zr, err := gzip.NewReader(bytes.NewReader(data))
	if err != nil {
		t.Fatalf("gzip reader: %v", err)
	}
	defer zr.Close()
	body, err := io.ReadAll(zr)
	if err != nil {
		t.Fatalf("gzip read: %v", err)
	}
	return body
}
