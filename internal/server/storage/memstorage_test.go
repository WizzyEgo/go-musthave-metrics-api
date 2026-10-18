package storage

import (
	"errors"
	"sync"
	"testing"

	"go-musthave-metrics-tpl/internal/model"
)

func TestMemStorageUpdateGauge(t *testing.T) {
	store := NewMemStorage()
	store.UpdateGauge("Alloc", 10.5)
	store.UpdateGauge("Alloc", 20.5)

	got, ok := store.GetGauge("Alloc")
	if !ok {
		t.Fatal("Alloc was not stored")
	}
	if got != 20.5 {
		t.Fatalf("Alloc = %v, want 20.5", got)
	}
}

func TestMemStorageUpdateCounter(t *testing.T) {
	store := NewMemStorage()
	store.UpdateCounter("PollCount", 1)
	store.UpdateCounter("PollCount", 4)

	got, ok := store.GetCounter("PollCount")
	if !ok {
		t.Fatal("PollCount was not stored")
	}
	if got != 5 {
		t.Fatalf("PollCount = %d, want 5", got)
	}
}

func TestMemStorageSetCounter(t *testing.T) {
	store := NewMemStorage()
	store.UpdateCounter("PollCount", 5)
	store.SetCounter("PollCount", 2)

	got, ok := store.GetCounter("PollCount")
	if !ok {
		t.Fatal("PollCount was not stored")
	}
	if got != 2 {
		t.Fatalf("PollCount = %d, want 2", got)
	}
}

func TestMemStorageGetUnknown(t *testing.T) {
	store := NewMemStorage()

	if _, ok := store.GetGauge("Alloc"); ok {
		t.Fatal("expected unknown gauge")
	}
	if _, ok := store.GetCounter("PollCount"); ok {
		t.Fatal("expected unknown counter")
	}
}

func TestMemStorageAll(t *testing.T) {
	store := NewMemStorage()
	store.UpdateGauge("Alloc", 1.5)
	store.UpdateCounter("PollCount", 2)

	gauges, counters := store.GetAll()
	if gauges["Alloc"] != 1.5 {
		t.Fatalf("GetAll gauges Alloc = %v, want 1.5", gauges["Alloc"])
	}
	if counters["PollCount"] != 2 {
		t.Fatalf("GetAll counters PollCount = %d, want 2", counters["PollCount"])
	}

	gauges["Alloc"] = 0
	counters["PollCount"] = 0
	if got, _ := store.GetGauge("Alloc"); got != 1.5 {
		t.Fatal("GetAll must return a copy of gauges")
	}
	if got, _ := store.GetCounter("PollCount"); got != 2 {
		t.Fatal("GetAll must return a copy of counters")
	}
}

func TestMemStorageUpdateBatch(t *testing.T) {
	store := NewMemStorage()
	store.UpdateCounter("PollCount", 1)

	value := 1.5
	first := int64(2)
	second := int64(3)
	updated, err := store.UpdateBatch([]model.Metrics{
		{ID: "Alloc", MType: model.Gauge, Value: &value},
		{ID: "PollCount", MType: model.Counter, Delta: &first},
		{ID: "PollCount", MType: model.Counter, Delta: &second},
	})
	if err != nil {
		t.Fatalf("UpdateBatch() error = %v", err)
	}
	if len(updated) != 3 || updated[0].Value == nil || *updated[0].Value != 1.5 {
		t.Fatalf("updated gauge = %+v", updated)
	}
	if updated[1].Delta == nil || *updated[1].Delta != 3 || updated[2].Delta == nil || *updated[2].Delta != 6 {
		t.Fatalf("updated counters = %+v %+v", updated[1], updated[2])
	}

	got, ok := store.GetCounter("PollCount")
	if !ok || got != 6 {
		t.Fatalf("PollCount = (%d, %v), want (6, true)", got, ok)
	}
}

func TestMemStorageUpdateBatchRejectsInvalidWithoutChanges(t *testing.T) {
	store := NewMemStorage()
	store.UpdateCounter("PollCount", 4)
	value := 1.0
	_, err := store.UpdateBatch([]model.Metrics{
		{ID: "Alloc", MType: model.Gauge, Value: &value},
		{ID: "Bad", MType: "unknown"},
	})
	if !errors.Is(err, ErrInvalidMetric) {
		t.Fatalf("UpdateBatch() error = %v, want ErrInvalidMetric", err)
	}
	if _, ok := store.GetGauge("Alloc"); ok {
		t.Fatal("invalid batch stored Alloc")
	}
	if got, _ := store.GetCounter("PollCount"); got != 4 {
		t.Fatalf("PollCount = %d, want 4", got)
	}
}

func TestMemStorageUpdateBatchConcurrent(t *testing.T) {
	store := NewMemStorage()
	const workers = 50
	var wg sync.WaitGroup
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			delta := int64(1)
			value := 1.0
			if _, err := store.UpdateBatch([]model.Metrics{
				{ID: "PollCount", MType: model.Counter, Delta: &delta},
				{ID: "Alloc", MType: model.Gauge, Value: &value},
			}); err != nil {
				t.Errorf("UpdateBatch() error = %v", err)
			}
			store.GetAll()
		}()
	}
	wg.Wait()

	got, ok := store.GetCounter("PollCount")
	if !ok || got != workers {
		t.Fatalf("PollCount = (%d, %v), want (%d, true)", got, ok, workers)
	}
}
