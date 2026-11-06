package middleware

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"go-musthave-metrics-tpl/internal/compress"
	"go-musthave-metrics-tpl/internal/hash"
)

func TestHashSHA256PassesThroughWithoutKey(t *testing.T) {
	var called bool
	h := HashSHA256("")(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodPost, "/update/", bytes.NewReader([]byte(`{"id":"Alloc"}`)))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if !called {
		t.Fatal("handler was not called")
	}
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	if rec.Header().Get(hash.Header) != "" {
		t.Fatalf("HashSHA256 = %q, want empty", rec.Header().Get(hash.Header))
	}
}

func TestHashSHA256AcceptsMissingHeader(t *testing.T) {
	const key = "secret"
	body := []byte(`{"id":"Alloc","type":"gauge","value":1}`)
	var gotBody []byte
	h := HashSHA256(key)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var err error
		gotBody, err = io.ReadAll(r.Body)
		if err != nil {
			t.Errorf("read body: %v", err)
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))

	req := httptest.NewRequest(http.MethodPost, "/update/", bytes.NewReader(body))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	if !bytes.Equal(gotBody, body) {
		t.Fatalf("handler body = %q, want %q", gotBody, body)
	}
	want := hash.Sum([]byte(`{"ok":true}`), key)
	if got := rec.Header().Get(hash.Header); got != want {
		t.Fatalf("response hash = %q, want %q", got, want)
	}
}

func TestHashSHA256RejectsMismatch(t *testing.T) {
	const key = "secret"
	var called bool
	h := HashSHA256(key)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodPost, "/update/", bytes.NewReader([]byte("body")))
	req.Header.Set(hash.Header, hash.Sum([]byte("other"), key))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if called {
		t.Fatal("handler was called for a bad signature")
	}
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
	if got := rec.Header().Get(hash.Header); got != hash.Sum(nil, key) {
		t.Fatalf("response hash = %q, want hash of empty body", got)
	}
}

func TestHashSHA256SignsPlainBodyUnderGzip(t *testing.T) {
	const key = "secret"
	plain := []byte(`{"id":"Alloc","type":"gauge","value":1}`)
	compressed, err := compress.Encode(plain)
	if err != nil {
		t.Fatalf("encode: %v", err)
	}

	var gotPlain []byte
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var readErr error
		gotPlain, readErr = io.ReadAll(r.Body)
		if readErr != nil {
			t.Errorf("read body: %v", readErr)
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(plain)
	})
	h := Gzip(HashSHA256(key)(inner))

	req := httptest.NewRequest(http.MethodPost, "/update/", bytes.NewReader(compressed))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Content-Encoding", "gzip")
	req.Header.Set("Accept-Encoding", "gzip")
	req.Header.Set(hash.Header, hash.Sum(plain, key))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	if !bytes.Equal(gotPlain, plain) {
		t.Fatalf("decompressed body = %q, want %q", gotPlain, plain)
	}
	if rec.Header().Get("Content-Encoding") != "gzip" {
		t.Fatalf("Content-Encoding = %q, want gzip", rec.Header().Get("Content-Encoding"))
	}
	if got := rec.Header().Get(hash.Header); got != hash.Sum(plain, key) {
		t.Fatalf("response hash = %q, want hash of plain body", got)
	}
}
