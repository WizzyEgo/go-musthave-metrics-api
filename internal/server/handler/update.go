package handler

import (
	"context"
	"errors"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"go.uber.org/zap"

	"go-musthave-metrics-tpl/internal/model"
	"go-musthave-metrics-tpl/internal/server/storage"
)

var errBadValue = errors.New("bad metric value")

func Update(log *zap.Logger, store storage.Storage) http.HandlerFunc {
	if log == nil {
		log = zap.NewNop()
	}
	return func(w http.ResponseWriter, r *http.Request) {
		metricType := chi.URLParam(r, "type")
		metricName := chi.URLParam(r, "name")
		metricValue := chi.URLParam(r, "value")

		if metricName == "" {
			w.WriteHeader(http.StatusNotFound)
			return
		}

		if metricType != model.Gauge && metricType != model.Counter {
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		if err := updateMetric(r.Context(), store, metricType, metricName, metricValue); err != nil {
			if errors.Is(err, errBadValue) {
				w.WriteHeader(http.StatusBadRequest)
				return
			}
			writeInternalError(log, w, "failed to update metric", err)
			return
		}

		w.Header().Set("Content-Type", "text/plain")
		w.WriteHeader(http.StatusOK)
	}
}

func updateMetric(ctx context.Context, store storage.Storage, metricType, metricName, metricValue string) error {
	switch metricType {
	case model.Gauge:
		val, err := strconv.ParseFloat(metricValue, 64)
		if err != nil {
			return errBadValue
		}
		return store.UpdateGauge(ctx, metricName, val)
	case model.Counter:
		val, err := strconv.ParseInt(metricValue, 10, 64)
		if err != nil {
			return errBadValue
		}
		return store.UpdateCounter(ctx, metricName, val)
	default:
		return errBadValue
	}
}
