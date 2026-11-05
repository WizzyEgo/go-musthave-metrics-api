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
	"go-musthave-metrics-tpl/internal/hash"
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

	s.workerPool(ctx, splitChunks(batch, s.workerCount()))
}

type chunkResult struct {
	metrics []model.Metrics
	err     error
}

// splitChunks делит метрики на limit непустых пачек для параллельной отправки.
func splitChunks(metrics []model.Metrics, limit int) [][]model.Metrics {
	if len(metrics) == 0 {
		return nil
	}
	if limit < 1 {
		limit = 1
	}
	if limit > len(metrics) {
		limit = len(metrics)
	}

	chunks := make([][]model.Metrics, limit)
	for i, metric := range metrics {
		chunks[i%limit] = append(chunks[i%limit], metric)
	}
	return chunks
}

// workerPool отправляет пачки пулом воркеров.
// Одновременно выполняется не больше rateLimit HTTP-запросов.
func (s *Service) workerPool(ctx context.Context, chunks [][]model.Metrics) {
	if len(chunks) == 0 {
		return
	}

	workers := s.workerCount()
	if workers > len(chunks) {
		workers = len(chunks)
	}

	// Буфер равен числу воркеров, чтобы фидер заполнил слоты сразу,
	// а воркеры забирали задачи, не дожидаясь следующего send.
	jobs := make(chan []model.Metrics, workers)
	results := make(chan chunkResult, len(chunks))

	var wg sync.WaitGroup
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for chunk := range jobs {
				results <- chunkResult{metrics: chunk, err: s.sendBatch(ctx, chunk)}
			}
		}()
	}

	go func() {
		defer close(jobs)
		for _, chunk := range chunks {
			select {
			case <-ctx.Done():
				return
			case jobs <- chunk:
			}
		}
	}()

	go func() {
		wg.Wait()
		close(results)
	}()

	for result := range results {
		if result.err != nil {
			if !errors.Is(result.err, context.Canceled) {
				s.log.Error("failed to send metrics", zap.Int("count", len(result.metrics)), zap.Error(result.err))
			}
			continue
		}
		// Вычитаем отправленную дельту, а не обнуляем счётчик:
		// за время запроса Collect мог увеличить его снова.
		for _, metric := range result.metrics {
			if metric.MType == model.Counter && metric.Delta != nil {
				s.metrics.SubCounter(metric.ID, *metric.Delta)
			}
		}
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

	// Подпись от несжатого JSON: её можно проверить по данным, а не по gzip-байтам.
	var sum string
	if s.key != "" {
		sum = hash.Sum(payload, s.key)
	}

	compressed, err := compress.Encode(payload)
	if err != nil {
		return err
	}
	return s.post(ctx, compressed, sum)
}

func (s *Service) post(ctx context.Context, body []byte, sum string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, s.serverURL+"/updates/", bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Content-Encoding", "gzip")
	if sum != "" {
		req.Header.Set(hash.Header, sum)
	}

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
	wg.Add(3)
	go func() {
		defer wg.Done()
		s.loop(ctx, s.pollInterval, s.Collect)
	}()
	go func() {
		defer wg.Done()
		// Первый сбор до тикера, чтобы метрики памяти и CPU успели
		// попасть в ближайшую отправку.
		s.collectSystem(ctx)
		s.loop(ctx, s.pollInterval, func() { s.collectSystem(ctx) })
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
