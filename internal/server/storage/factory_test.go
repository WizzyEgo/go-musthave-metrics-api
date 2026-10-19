package storage

import (
	"context"
	"path/filepath"
	"testing"

	"go.uber.org/zap"
)

func TestNewFallsBackToFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "metrics.json")
	store, file, err := New(zap.NewNop(), "", nil, path, true, false)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	if file == nil {
		t.Fatal("file storage is nil")
	}

	store.UpdateGauge(context.Background(), "Alloc", 42)
	if err := file.Save(); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	restored, _, err := New(zap.NewNop(), "", nil, path, true, true)
	if err != nil {
		t.Fatalf("restore New() error = %v", err)
	}
	value, ok := restored.GetGauge(context.Background(), "Alloc")
	if !ok || value != 42 {
		t.Fatalf("GetGauge() = (%v, %v), want (42, true)", value, ok)
	}
}

func TestNewFallsBackToMemoryWhenFilePathEmpty(t *testing.T) {
	store, file, err := New(zap.NewNop(), "", nil, "", true, true)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	if file == nil {
		t.Fatal("file storage is nil")
	}

	store.UpdateCounter(context.Background(), "PollCount", 3)
	value, ok := store.GetCounter(context.Background(), "PollCount")
	if !ok || value != 3 {
		t.Fatalf("GetCounter() = (%v, %v), want (3, true)", value, ok)
	}
	if err := file.Save(); err != nil {
		t.Fatalf("Save() error = %v", err)
	}
}

func TestNewPostgresRequiresDB(t *testing.T) {
	_, _, err := New(zap.NewNop(), "postgres://localhost/metrics", nil, "", true, false)
	if err == nil {
		t.Fatal("expected error for missing database")
	}
}

func TestNewTreatsBlankDSNAsMissing(t *testing.T) {
	store, file, err := New(zap.NewNop(), "   ", nil, "", true, false)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	if file == nil || store == nil {
		t.Fatal("expected memory fallback for blank DSN")
	}
}
