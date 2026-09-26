package logger

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest/observer"
)

func TestInitialize(t *testing.T) {
	prev := Log
	t.Cleanup(func() { Log = prev })

	if err := Initialize("no-such-level"); err == nil {
		t.Fatal("expected error for unknown level")
	}
	if err := Initialize("info"); err != nil {
		t.Fatalf("Initialize() error = %v", err)
	}
	if Log == prev {
		t.Fatal("Log was not replaced")
	}
}

func TestRequestLoggerRecordsRequestAndResponse(t *testing.T) {
	core, recorded := observer.New(zapcore.InfoLevel)
	prev := Log
	Log = zap.New(core)
	t.Cleanup(func() { Log = prev })

	const uri = "/update/gauge/SingletonAlloc/1.5"
	body := []byte("hello")
	handler := RequestLogger(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write(body)
	}))

	req := httptest.NewRequest(http.MethodPost, uri, nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusCreated)
	}
	if rec.Body.String() != string(body) {
		t.Fatalf("body = %q, want %q", rec.Body.String(), body)
	}

	fields := logFields(recorded.All(), uri)
	if fields == nil {
		t.Fatal("no log entry for request")
	}
	if fields["method"] != http.MethodPost {
		t.Fatalf("method = %v, want POST", fields["method"])
	}
	duration, ok := fields["duration"].(time.Duration)
	if !ok || duration < 0 {
		t.Fatalf("duration = %v (%T), want non-negative time.Duration", fields["duration"], fields["duration"])
	}
	if fields["status"] != int64(http.StatusCreated) {
		t.Fatalf("status = %v, want %d", fields["status"], http.StatusCreated)
	}
	if fields["size"] != int64(len(body)) {
		t.Fatalf("size = %v, want %d", fields["size"], len(body))
	}
}

func TestRequestLoggerDefaultsStatusWhenHeaderIsOmitted(t *testing.T) {
	core, recorded := observer.New(zapcore.InfoLevel)
	prev := Log
	Log = zap.New(core)
	t.Cleanup(func() { Log = prev })

	const uri = "/singleton-default-status"
	handler := RequestLogger(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("ok"))
	}))

	req := httptest.NewRequest(http.MethodGet, uri, nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}

	fields := logFields(recorded.All(), uri)
	if fields == nil {
		t.Fatal("no log entry for request")
	}
	if fields["status"] != int64(http.StatusOK) {
		t.Fatalf("logged status = %v, want %d", fields["status"], http.StatusOK)
	}
	if fields["size"] != int64(2) {
		t.Fatalf("size = %v, want 2", fields["size"])
	}
}

func logFields(entries []observer.LoggedEntry, uri string) map[string]any {
	for i := len(entries) - 1; i >= 0; i-- {
		fields := entries[i].ContextMap()
		if fields["uri"] == uri {
			return fields
		}
	}
	return nil
}
