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
	"go-musthave-metrics-tpl/internal/retry"
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
	s.reportMu.Lock()
	defer s.reportMu.Unlock()

	gauges, counters := s.metrics.Snapshot()
	batch := make([]model.Metrics, 0, len(gauges)+len(counters))
	for name, value := range gauges {
		v := value
		batch = append(batch, model.Metrics{
			ID:    name,
			MType: model.Gauge,
			Value: &v,
		})
	}
	for name, value := range counters {
		delta := value
		batch = append(batch, model.Metrics{
			ID:    name,
			MType: model.Counter,
			Delta: &delta,
		})
	}
	if len(batch) == 0 {
		return
	}

	if err := s.sendBatch(ctx, batch); err != nil {
		if !errors.Is(err, context.Canceled) {
			s.log.Error("failed to send metrics", zap.Int("count", len(batch)), zap.Error(err))
		}
		return
	}

	// Вычитаем отправленную дельту, а не обнуляем счётчик:
	// за время запроса Collect мог увеличить его снова.
	for name, value := range counters {
		s.metrics.SubCounter(name, value)
	}
}

func (s *Service) sendBatch(ctx context.Context, metrics []model.Metrics) error {
	if len(metrics) == 0 {
		return nil
	}

	payload, err := easyjson.Marshal(model.MetricsList(metrics))
	if err != nil {
		return err
	}

	compressed, err := compress.Encode(payload)
	if err != nil {
		return err
	}

	attempt := 0
	return retry.Do(ctx, func() error {
		attempt++
		err := s.post(ctx, compressed)
		if !retry.Connection(err) {
			return err
		}
		s.log.Warn("retriable connection error", zap.Int("attempt", attempt), zap.Error(err))
		return retry.Retriable(err)
	})
}

func (s *Service) post(ctx context.Context, body []byte) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, s.serverURL+"/updates/", bytes.NewReader(body))
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
