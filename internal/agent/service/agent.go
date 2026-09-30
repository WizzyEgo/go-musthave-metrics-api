package service

import (
	"bytes"
	"compress/gzip"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"runtime"
	"time"

	"github.com/mailru/easyjson"

	"go-musthave-metrics-tpl/internal/server/model"
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
		v := value
		_ = s.sendMetric(model.Metrics{
			ID:    name,
			MType: model.Gauge,
			Value: &v,
		})
	}

	for name, value := range counters {
		delta := value
		if err := s.sendMetric(model.Metrics{
			ID:    name,
			MType: model.Counter,
			Delta: &delta,
		}); err == nil {
			s.metrics.SubCounter(name, value)
		}
	}
}

func (s *Service) sendMetric(metric model.Metrics) error {
	payload, err := easyjson.Marshal(metric)
	if err != nil {
		return err
	}

	var buf bytes.Buffer
	zw, err := gzip.NewWriterLevel(&buf, gzip.BestSpeed)
	if err != nil {
		return err
	}
	if _, err = zw.Write(payload); err != nil {
		return err
	}
	if err = zw.Close(); err != nil {
		return err
	}

	req, err := http.NewRequest(http.MethodPost, s.serverURL+"/update", &buf)
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Content-Encoding", "gzip")

	resp, err := s.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	_, _ = io.Copy(io.Discard, resp.Body)

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}
	return nil
}

func (s *Service) Run() {
	go func() {
		for {
			time.Sleep(s.pollInterval)
			s.Collect()
		}
	}()

	for {
		time.Sleep(s.reportInterval)
		s.Report()
	}
}
