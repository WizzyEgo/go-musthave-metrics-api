package storage

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"go.uber.org/zap"

	"go-musthave-metrics-tpl/internal/model"
)

func TestFileStorageSyncRoundTrip(t *testing.T) {
	path := filepath.Join(t.TempDir(), "metrics.json")
	store := mustFileStorage(t, path, true, false)
	store.UpdateGauge("Alloc", 1.5)
	store.UpdateGauge("Alloc", 2.5)
	store.UpdateCounter("PollCount", 2)
	store.UpdateCounter("PollCount", 3)
	store.UpdateGauge("ZeroGauge", 0)
	store.UpdateCounter("ZeroCounter", 0)

	restored := mustFileStorage(t, path, false, true)

	gauge, ok := restored.GetGauge("Alloc")
	if !ok || gauge != 2.5 {
		t.Fatalf("Alloc = %v, %v, want 2.5", gauge, ok)
	}
	counter, ok := restored.GetCounter("PollCount")
	if !ok || counter != 5 {
		t.Fatalf("PollCount = %d, %v, want 5", counter, ok)
	}
	gauge, ok = restored.GetGauge("ZeroGauge")
	if !ok || gauge != 0 {
		t.Fatalf("ZeroGauge = %v, %v, want 0", gauge, ok)
	}
	counter, ok = restored.GetCounter("ZeroCounter")
	if !ok || counter != 0 {
		t.Fatalf("ZeroCounter = %d, %v, want 0", counter, ok)
	}

	if err := restored.Load(); err != nil {
		t.Fatalf("second Load() error = %v", err)
	}
	counter, _ = restored.GetCounter("PollCount")
	if counter != 5 {
		t.Fatalf("PollCount after second load = %d, want 5", counter)
	}
}

func TestFileStorageAsyncSave(t *testing.T) {
	path := filepath.Join(t.TempDir(), "nested", "metrics.json")
	store := mustFileStorage(t, path, false, false)
	store.UpdateGauge("LastGC", 12)
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatalf("file exists before Save, err = %v", err)
	}
	if err := store.Save(); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	metrics := readMetricsFile(t, path)
	if len(metrics) != 1 || metrics[0].ID != "LastGC" || metrics[0].MType != model.Gauge || metrics[0].Value == nil || *metrics[0].Value != 12 {
		t.Fatalf("metrics = %+v", metrics)
	}
}

func TestFileStorageLoadArrayAndNDJSON(t *testing.T) {
	dir := t.TempDir()
	arrayPath := filepath.Join(dir, "array.json")
	array := []byte(`[
  {"id":"LastGC","type":"gauge","value":1257894000000000000},
  {"id":"NumGC","type":"counter","delta":42}
]`)
	if err := os.WriteFile(arrayPath, array, 0644); err != nil {
		t.Fatal(err)
	}
	store := mustFileStorage(t, arrayPath, false, true)
	if value, ok := store.GetCounter("NumGC"); !ok || value != 42 {
		t.Fatalf("NumGC = %d, %v", value, ok)
	}
	if _, ok := store.GetGauge("LastGC"); !ok {
		t.Fatal("LastGC was not loaded")
	}

	ndjsonPath := filepath.Join(dir, "lines.json")
	ndjson := []byte("{\"id\":\"Alloc\",\"type\":\"gauge\",\"value\":1.25}\n{\"id\":\"PollCount\",\"type\":\"counter\",\"delta\":7}\n")
	if err := os.WriteFile(ndjsonPath, ndjson, 0644); err != nil {
		t.Fatal(err)
	}
	lines := mustFileStorage(t, ndjsonPath, false, true)
	if value, ok := lines.GetGauge("Alloc"); !ok || value != 1.25 {
		t.Fatalf("Alloc = %v, %v", value, ok)
	}
	if value, ok := lines.GetCounter("PollCount"); !ok || value != 7 {
		t.Fatalf("PollCount = %d, %v", value, ok)
	}
}

func TestFileStorageLoadMissingAndEmpty(t *testing.T) {
	mustFileStorage(t, filepath.Join(t.TempDir(), "missing.json"), false, true)

	emptyPath := filepath.Join(t.TempDir(), "empty.json")
	if err := os.WriteFile(emptyPath, []byte(" \n"), 0644); err != nil {
		t.Fatal(err)
	}
	mustFileStorage(t, emptyPath, false, true)
}

func TestFileStorageLoadInvalid(t *testing.T) {
	path := filepath.Join(t.TempDir(), "bad.json")
	if err := os.WriteFile(path, []byte("{"), 0644); err != nil {
		t.Fatal(err)
	}
	if _, err := NewFileStorage(zap.NewNop(), path, false, true); err == nil {
		t.Fatal("expected decode error")
	}
}

func TestFileStorageEmptyPath(t *testing.T) {
	store := mustFileStorage(t, "", true, true)
	store.UpdateGauge("Alloc", 1)
	if err := store.Load(); err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if err := store.Save(); err != nil {
		t.Fatalf("Save() error = %v", err)
	}
	if _, ok := store.GetGauge("Alloc"); !ok {
		t.Fatal("Alloc was not stored in memory")
	}
}

func TestFileStorageFlushEvery(t *testing.T) {
	path := filepath.Join(t.TempDir(), "metrics.json")
	store := mustFileStorage(t, path, false, false)
	store.UpdateGauge("Alloc", 3)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go store.FlushEvery(ctx, 20*time.Millisecond)

	deadline := time.Now().Add(time.Second)
	for time.Now().Before(deadline) {
		data, err := os.ReadFile(path)
		if err == nil {
			var metrics []model.Metrics
			if json.Unmarshal(data, &metrics) == nil &&
				len(metrics) == 1 &&
				metrics[0].ID == "Alloc" &&
				metrics[0].Value != nil &&
				*metrics[0].Value == 3 {
				return
			}
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatal("metrics were not flushed")
}

func mustFileStorage(t *testing.T, path string, synchronous, restore bool) *FileStorage {
	t.Helper()
	store, err := NewFileStorage(zap.NewNop(), path, synchronous, restore)
	if err != nil {
		t.Fatalf("NewFileStorage() error = %v", err)
	}
	return store
}

func readMetricsFile(t *testing.T, path string) []model.Metrics {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	var metrics []model.Metrics
	if err := json.Unmarshal(data, &metrics); err != nil {
		t.Fatalf("unmarshal %s: %v\n%s", path, err, data)
	}
	return metrics
}
