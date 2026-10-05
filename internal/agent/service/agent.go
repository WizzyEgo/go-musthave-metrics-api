package service

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"runtime"
	"sync"
	"time"

	"github.com/mailru/easyjson"
	"go.uber.org/zap"

	"go-musthave-metrics-tpl/internal/compress"
	"go-musthave-metrics-tpl/internal/model"
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
	s.report(context.Background())
}

func (s *Service) report(ctx context.Context) {
	gauges, counters := s.metrics.Snapshot()

	for name, value := range gauges {
		if ctx.Err() != nil {
			return
		}
		v := value
		if err := s.sendMetric(ctx, model.Metrics{
			ID:    name,
			MType: model.Gauge,
			Value: &v,
		}); err != nil {
			s.logSendError(model.Gauge, name, err)
		}
	}

	for name, value := range counters {
		if ctx.Err() != nil {
			return
		}
		delta := value
		if err := s.sendMetric(ctx, model.Metrics{
			ID:    name,
			MType: model.Counter,
			Delta: &delta,
		}); err != nil {
			s.logSendError(model.Counter, name, err)
			continue
		}
		s.metrics.SubCounter(name, value)
	}
}

func (s *Service) logSendError(metricType, name string, err error) {
	if errors.Is(err, context.Canceled) {
		return
	}
	s.log.Error("failed to send metric",
		zap.String("type", metricType),
		zap.String("id", name),
		zap.Error(err),
	)
}

func (s *Service) sendMetric(ctx context.Context, metric model.Metrics) error {
	payload, err := easyjson.Marshal(metric)
	if err != nil {
		return err
	}

	compressed, err := compress.Encode(payload)
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, s.serverURL+"/update", bytes.NewReader(compressed))
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

func (s *Service) Run(ctx context.Context) {
	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		s.loop(ctx, s.pollInterval, s.Collect)
	}()
	go func() {
		defer wg.Done()
		s.loop(ctx, s.reportInterval, func() { s.report(ctx) })
	}()
	wg.Wait()
}

func (s *Service) loop(ctx context.Context, interval time.Duration, fn func()) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			fn()
		}
	}
}
