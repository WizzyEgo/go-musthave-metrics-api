package server

type MemStorage struct {
	gauges   map[string]float64
	counters map[string]int64
}

func NewMemStorage() *MemStorage {
	return &MemStorage{
		gauges:   make(map[string]float64),
		counters: make(map[string]int64),
	}
}

func (ms *MemStorage) UpdateGauge(name string, value float64) {
	ms.gauges[name] = value
}

func (ms *MemStorage) UpdateCounter(name string, value int64) {
	ms.counters[name] += value
}

func (ms *MemStorage) GetGauge(name string) (float64, bool) {
	value, ok := ms.gauges[name]
	return value, ok
}

func (ms *MemStorage) GetCounter(name string) (int64, bool) {
	value, ok := ms.counters[name]
	return value, ok
}

func (ms *MemStorage) GetAll() (map[string]float64, map[string]int64) {
	gauges := make(map[string]float64, len(ms.gauges))
	for name, value := range ms.gauges {
		gauges[name] = value
	}

	counters := make(map[string]int64, len(ms.counters))
	for name, value := range ms.counters {
		counters[name] = value
	}

	return gauges, counters
}
