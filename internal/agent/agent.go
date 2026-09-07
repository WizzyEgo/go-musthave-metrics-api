package agent

import (
	"fmt"
	"io"
	"log"
	"math/rand"
	"net/http"
	"runtime"
	"strconv"
	"time"
)

func (s *Service) Collect() {
	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)

	s.metrics.SetGauge("Alloc", float64(memStats.Alloc))
	s.metrics.SetGauge("BuckHashSys", float64(memStats.BuckHashSys))
	s.metrics.SetGauge("Frees", float64(memStats.Frees))
	s.metrics.SetGauge("GCCPUFraction", memStats.GCCPUFraction)
	s.metrics.SetGauge("GCSys", float64(memStats.GCSys))
	s.metrics.SetGauge("HeapAlloc", float64(memStats.HeapAlloc))
	s.metrics.SetGauge("HeapIdle", float64(memStats.HeapIdle))
	s.metrics.SetGauge("HeapInuse", float64(memStats.HeapInuse))
	s.metrics.SetGauge("HeapObjects", float64(memStats.HeapObjects))
	s.metrics.SetGauge("HeapReleased", float64(memStats.HeapReleased))
	s.metrics.SetGauge("HeapSys", float64(memStats.HeapSys))
	s.metrics.SetGauge("LastGC", float64(memStats.LastGC))
	s.metrics.SetGauge("Lookups", float64(memStats.Lookups))
	s.metrics.SetGauge("MCacheInuse", float64(memStats.MCacheInuse))
	s.metrics.SetGauge("MCacheSys", float64(memStats.MCacheSys))
	s.metrics.SetGauge("MSpanInuse", float64(memStats.MSpanInuse))
	s.metrics.SetGauge("MSpanSys", float64(memStats.MSpanSys))
	s.metrics.SetGauge("Mallocs", float64(memStats.Mallocs))
	s.metrics.SetGauge("NextGC", float64(memStats.NextGC))
	s.metrics.SetGauge("NumForcedGC", float64(memStats.NumForcedGC))
	s.metrics.SetGauge("NumGC", float64(memStats.NumGC))
	s.metrics.SetGauge("OtherSys", float64(memStats.OtherSys))
	s.metrics.SetGauge("PauseTotalNs", float64(memStats.PauseTotalNs))
	s.metrics.SetGauge("StackInuse", float64(memStats.StackInuse))
	s.metrics.SetGauge("StackSys", float64(memStats.StackSys))
	s.metrics.SetGauge("Sys", float64(memStats.Sys))
	s.metrics.SetGauge("TotalAlloc", float64(memStats.TotalAlloc))
	s.metrics.SetGauge("RandomValue", rand.Float64())
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
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			log.Fatalf("failed to close response body: %s", err)
		}
	}(resp.Body)

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
