package server

import "testing"

func TestMemStorageUpdateGauge(t *testing.T) {
	store := NewMemStorage()
	store.UpdateGauge("Alloc", 10.5)
	store.UpdateGauge("Alloc", 20.5)

	if got := store.gauges["Alloc"]; got != 20.5 {
		t.Fatalf("Alloc = %v, want 20.5", got)
	}
}

func TestMemStorageUpdateCounter(t *testing.T) {
	store := NewMemStorage()
	store.UpdateCounter("PollCount", 1)
	store.UpdateCounter("PollCount", 4)

	if got := store.counters["PollCount"]; got != 5 {
		t.Fatalf("PollCount = %d, want 5", got)
	}
}
