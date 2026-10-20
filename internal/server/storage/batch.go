package storage

import (
	"errors"

	"go-musthave-metrics-tpl/internal/model"
)

var (
	ErrInvalidMetric = errors.New("invalid metric")
	ErrEmptyMetricID = errors.New("empty metric id")
)

func validateMetrics(metrics []model.Metrics) error {
	for i := range metrics {
		if err := validateMetric(metrics[i]); err != nil {
			return err
		}
	}
	return nil
}

func validateMetric(metric model.Metrics) error {
	if metric.ID == "" {
		return ErrEmptyMetricID
	}
	switch metric.MType {
	case model.Gauge:
		if metric.Value == nil {
			return ErrInvalidMetric
		}
	case model.Counter:
		if metric.Delta == nil {
			return ErrInvalidMetric
		}
	default:
		return ErrInvalidMetric
	}
	return nil
}

func float64Ptr(v float64) *float64 {
	return &v
}

func int64Ptr(v int64) *int64 {
	return &v
}

func metricFromStored(metric model.Metrics, value float64, delta int64) model.Metrics {
	updated := model.Metrics{
		ID:    metric.ID,
		MType: metric.MType,
		Hash:  metric.Hash,
	}
	switch metric.MType {
	case model.Gauge:
		updated.Value = float64Ptr(value)
	case model.Counter:
		updated.Delta = int64Ptr(delta)
	}
	return updated
}
