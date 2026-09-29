package middleware

import (
	"bytes"
	"compress/gzip"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestGzipCompressesJSONAndHTML(t *testing.T) {
	payload := []byte(`{"id":"Alloc","type":"gauge","value":1}`)
	h := Gzip(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", r.Header.Get("X-Response-Type"))
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(payload)
	}))

	tests := []struct {
		name        string
		contentType string
	}{
		{name: "json", contentType: "application/json"},
		{name: "json with charset", contentType: "application/json; charset=utf-8"},
		{name: "html", contentType: "text/html; charset=utf-8"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			req.Header.Set("Accept-Encoding", "gzip")
			req.Header.Set("X-Response-Type", tt.contentType)
			rec := httptest.NewRecorder()
			h.ServeHTTP(rec, req)

			if rec.Code != http.StatusOK {
				t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
			}
			if got := rec.Header().Get("Content-Encoding"); got != "gzip" {
				t.Fatalf("Content-Encoding = %q, want gzip", got)
			}
			if got := rec.Header().Get("Content-Type"); got != tt.contentType {
				t.Fatalf("Content-Type = %q, want %q", got, tt.contentType)
			}

			body := gunzip(t, rec.Body.Bytes())
			if !bytes.Equal(body, payload) {
				t.Fatalf("body = %q, want %q", body, payload)
			}
		})
	}
}

func TestGzipSkipsOtherContentTypes(t *testing.T) {
	const payload = "42.5"
	h := Gzip(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(payload))
	}))

	req := httptest.NewRequest(http.MethodGet, "/value/gauge/Alloc", nil)
	req.Header.Set("Accept-Encoding", "gzip, deflate")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Header().Get("Content-Encoding") != "" {
		t.Fatalf("Content-Encoding = %q, want empty", rec.Header().Get("Content-Encoding"))
	}
	if rec.Body.String() != payload {
		t.Fatalf("body = %q, want %q", rec.Body.String(), payload)
	}
}

func TestGzipLeavesResponseUncompressedWithoutAcceptEncoding(t *testing.T) {
	const payload = `{"ok":true}`
	h := Gzip(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(payload))
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Header().Get("Content-Encoding") != "" {
		t.Fatalf("Content-Encoding = %q, want empty", rec.Header().Get("Content-Encoding"))
	}
	if rec.Body.String() != payload {
		t.Fatalf("body = %q, want %q", rec.Body.String(), payload)
	}
}

func TestGzipDecompressesRequest(t *testing.T) {
	const payload = `{"id":"PollCount","type":"counter","delta":3}`
	var seen string

	h := Gzip(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Errorf("read body: %v", err)
		}
		seen = string(body)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(body)
	}))

	var buf bytes.Buffer
	zw, err := gzip.NewWriterLevel(&buf, gzip.BestSpeed)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = zw.Write([]byte(payload)); err != nil {
		t.Fatal(err)
	}
	if err = zw.Close(); err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest(http.MethodPost, "/update", &buf)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Content-Encoding", "gzip")
	req.Header.Set("Accept-Encoding", "gzip")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	if seen != payload {
		t.Fatalf("handler saw %q, want %q", seen, payload)
	}
	if got := string(gunzip(t, rec.Body.Bytes())); got != payload {
		t.Fatalf("response = %q, want %q", got, payload)
	}
}

func TestGzipRejectsInvalidBody(t *testing.T) {
	h := Gzip(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("handler must not be called for invalid gzip")
	}))

	req := httptest.NewRequest(http.MethodPost, "/update", strings.NewReader("not-gzip"))
	req.Header.Set("Content-Encoding", "gzip")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
}

func gunzip(t *testing.T, data []byte) []byte {
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
