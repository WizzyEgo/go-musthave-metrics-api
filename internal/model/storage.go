package model

type Storage interface {
	UpdateGauge(name string, value float64)
	UpdateCounter(name string, value int64)
}

func (ms *MemStorage) UpdateGauge(name string, value float64) {
	ms.gauges[name] = value
}

func (ms *MemStorage) UpdateCounter(name string, value int64) {
	ms.counters[name] = ms.counters[name] + value
}
