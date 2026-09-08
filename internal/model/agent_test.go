package model

import "testing"

func TestSetGaugeAndAddCounter(t *testing.T) {
	a := NewAgent()

	a.SetGauge("Alloc", 10.5)
	a.AddCounter("PollCount", 1)
	a.AddCounter("PollCount", 2)

	if got, ok := a.Gauge("Alloc"); !ok || got != 10.5 {
		t.Fatalf("Gauge(Alloc) = (%v, %v), want (10.5, true)", got, ok)
	}
	if got, ok := a.Counter("PollCount"); !ok || got != 3 {
		t.Fatalf("Counter(PollCount) = (%v, %v), want (3, true)", got, ok)
	}

	a.SubCounter("PollCount", 3)
	if got, _ := a.Counter("PollCount"); got != 0 {
		t.Fatalf("Counter(PollCount) after SubCounter = %d, want 0", got)
	}
}

func TestSnapshot(t *testing.T) {
	a := NewAgent()
	a.SetGauge("RandomValue", 0.42)
	a.AddCounter("PollCount", 5)

	gauges, counters := a.Snapshot()
	if gauges["RandomValue"] != 0.42 {
		t.Fatalf("snapshot gauge = %v, want 0.42", gauges["RandomValue"])
	}
	if counters["PollCount"] != 5 {
		t.Fatalf("snapshot counter = %d, want 5", counters["PollCount"])
	}

	gauges["RandomValue"] = 1
	counters["PollCount"] = 0
	if got, _ := a.Gauge("RandomValue"); got != 0.42 {
		t.Fatal("snapshot must return a copy of gauges")
	}
	if got, _ := a.Counter("PollCount"); got != 5 {
		t.Fatal("snapshot must return a copy of counters")
	}
}
