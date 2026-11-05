package handler

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/mailru/easyjson"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest/observer"

	"go-musthave-metrics-tpl/internal/compress"
	"go-musthave-metrics-tpl/internal/hash"
	"go-musthave-metrics-tpl/internal/model"
)

func TestRouterRejectsMismatchedHash(t *testing.T) {
	store := newStubStorage()
	h := newRouter(nil, store, "secret")

	body := []byte(`{"id":"Alloc","type":"gauge","value":1}`)
	req := httptest.NewRequest(http.MethodPost, "/update/", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set(hash.Header, hash.Sum([]byte("other"), "secret"))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
	if _, ok := store.gauges["Alloc"]; ok {
		t.Fatal("metric was stored despite a bad signature")
	}
}

func TestRouterAcceptsUnsignedRequestWhenKeyIsSet(t *testing.T) {
	store := newStubStorage()
	h := newRouter(nil, store, "secret")

	body := []byte(`{"id":"Alloc","type":"gauge","value":3.5}`)
	req := httptest.NewRequest(http.MethodPost, "/update/", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	if _, ok := store.gauges["Alloc"]; !ok {
		t.Fatal("metric was not stored")
	}
	var metric model.Metrics
	if err := easyjson.Unmarshal(rec.Body.Bytes(), &metric); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if got := rec.Header().Get(hash.Header); got != hash.Sum(rec.Body.Bytes(), "secret") {
		t.Fatalf("response hash = %q, want HMAC of the response body", got)
	}
}

func TestRouterChecksHashOfPlainBodyUnderGzip(t *testing.T) {
	store := newStubStorage()
	h := newRouter(nil, store, "secret")

	plain := []byte(`{"id":"Alloc","type":"gauge","value":7}`)
	compressed, err := compress.Encode(plain)
	if err != nil {
		t.Fatalf("encode: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/update/", bytes.NewReader(compressed))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Content-Encoding", "gzip")
	req.Header.Set("Accept-Encoding", "gzip")
	req.Header.Set(hash.Header, hash.Sum(plain, "secret"))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	if _, ok := store.gauges["Alloc"]; !ok {
		t.Fatal("metric was not stored")
	}
	if rec.Header().Get("Content-Encoding") != "gzip" {
		t.Fatalf("Content-Encoding = %q, want gzip", rec.Header().Get("Content-Encoding"))
	}
	zr, err := compress.NewReader(bytes.NewReader(rec.Body.Bytes()))
	if err != nil {
		t.Fatalf("gzip reader: %v", err)
	}
	decoded, err := io.ReadAll(zr)
	if err != nil {
		t.Fatalf("gzip read: %v", err)
	}
	if err = zr.Close(); err != nil {
		t.Fatalf("gzip close: %v", err)
	}
	if got := rec.Header().Get(hash.Header); got != hash.Sum(decoded, "secret") {
		t.Fatalf("response hash = %q, want HMAC of the plain response", got)
	}
	if got := rec.Header().Get(hash.Header); got == hash.Sum(rec.Body.Bytes(), "secret") {
		t.Fatal("response hash matches the compressed body")
	}
}

func TestRouterWithKeyKeepsRequestLog(t *testing.T) {
	core, recorded := observer.New(zapcore.InfoLevel)
	store := newStubStorage()
	h := newRouter(zap.New(core), store, "secret")

	req := httptest.NewRequest(http.MethodPost, "/update/gauge/Alloc/1", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	if rec.Header().Get(hash.Header) == "" {
		t.Fatal("response hash was not set")
	}
	fields := logFields(recorded.All(), "/update/gauge/Alloc/1")
	if fields == nil {
		t.Fatal("request was not logged")
	}
	if fields["status"] != int64(http.StatusOK) {
		t.Fatalf("logged status = %v, want %d", fields["status"], http.StatusOK)
	}
}
