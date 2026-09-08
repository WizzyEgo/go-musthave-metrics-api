package server

import "testing"

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
