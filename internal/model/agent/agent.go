package agent

import "sync"

type Agent struct {
	mu       sync.Mutex
	gauges   map[string]float64
	counters map[string]int64
}

func New() *Agent {
	return &Agent{
		gauges:   make(map[string]float64),
		counters: make(map[string]int64),
	}
}

func (a *Agent) SetGauge(name string, value float64) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.gauges[name] = value
}

func (a *Agent) AddCounter(name string, delta int64) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.counters[name] += delta
}

func (a *Agent) SubCounter(name string, delta int64) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.counters[name] -= delta
}

func (a *Agent) Gauge(name string) (float64, bool) {
	a.mu.Lock()
	defer a.mu.Unlock()
	v, ok := a.gauges[name]
	return v, ok
}

func (a *Agent) Counter(name string) (int64, bool) {
	a.mu.Lock()
	defer a.mu.Unlock()
	v, ok := a.counters[name]
	return v, ok
}

func (a *Agent) Snapshot() (map[string]float64, map[string]int64) {
	a.mu.Lock()
	defer a.mu.Unlock()

	gauges := make(map[string]float64, len(a.gauges))
	for name, value := range a.gauges {
		gauges[name] = value
	}

	counters := make(map[string]int64, len(a.counters))
	for name, value := range a.counters {
		counters[name] = value
	}

	return gauges, counters
}
