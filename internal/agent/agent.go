package agent

import (
	"fmt"
	"math/rand"
	"net/http"
	"runtime"
	"strconv"
	"time"
)

func (s *Service) collect() map[string]float64 {
	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)

	return map[string]float64{
		"Alloc":         float64(memStats.Alloc),
		"BuckHashSys":   float64(memStats.BuckHashSys),
		"Frees":         float64(memStats.Frees),
		"GCCPUFraction": memStats.GCCPUFraction,
		"GCSys":         float64(memStats.GCSys),
		"HeapAlloc":     float64(memStats.HeapAlloc),
		"HeapIdle":      float64(memStats.HeapIdle),
		"HeapInuse":     float64(memStats.HeapInuse),
		"HeapObjects":   float64(memStats.HeapObjects),
		"HeapReleased":  float64(memStats.HeapReleased),
		"HeapSys":       float64(memStats.HeapSys),
		"LastGC":        float64(memStats.LastGC),
		"Lookups":       float64(memStats.Lookups),
		"MCacheInuse":   float64(memStats.MCacheInuse),
		"MCacheSys":     float64(memStats.MCacheSys),
		"MSpanInuse":    float64(memStats.MSpanInuse),
		"MSpanSys":      float64(memStats.MSpanSys),
		"Mallocs":       float64(memStats.Mallocs),
		"NextGC":        float64(memStats.NextGC),
		"NumForcedGC":   float64(memStats.NumForcedGC),
		"NumGC":         float64(memStats.NumGC),
		"OtherSys":      float64(memStats.OtherSys),
		"PauseTotalNs":  float64(memStats.PauseTotalNs),
		"StackInuse":    float64(memStats.StackInuse),
		"StackSys":      float64(memStats.StackSys),
		"Sys":           float64(memStats.Sys),
		"TotalAlloc":    float64(memStats.TotalAlloc),
		"RandomValue":   rand.Float64(),
	}
}

func (s *Service) Collect() {
	for name, value := range s.collect() {
		s.metrics.SetGauge(name, value)
	}
	s.metrics.AddCounter("PollCount", 1)
}

func (s *Service) Report() {
	gauges, counters := s.metrics.Snapshot()

	for name, value := range gauges {
		_ = s.sendMetric("gauge", name, strconv.FormatFloat(value, 'f', -1, 64))
	}

	for name, value := range counters {
		if err := s.sendMetric("counter", name, strconv.FormatInt(value, 10)); err == nil {
			s.metrics.SubCounter(name, value)
		}
	}
}

func (s *Service) sendMetric(metricType, name, value string) error {
	url := fmt.Sprintf("%s/update/%s/%s/%s", s.serverURL, metricType, name, value)

	req, err := http.NewRequest(http.MethodPost, url, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "text/plain")

	resp, err := s.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}
	return nil
}

func (s *Service) Run() {
	go func() {
		for {
			time.Sleep(time.Duration(s.pollSec) * time.Second)
			s.Collect()
		}
	}()

	for {
		time.Sleep(time.Duration(s.reportSec) * time.Second)
		s.Report()
	}
}
