package handler

import (
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	"go-musthave-metrics-tpl/internal/model"
	"go-musthave-metrics-tpl/internal/model/server"
)

func Update(store server.Storage) http.HandlerFunc {
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

		if err := updateMetric(store, metricType, metricName, metricValue); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		w.Header().Set("Content-Type", "text/plain")
		w.WriteHeader(http.StatusOK)
	}
}

func updateMetric(store server.Storage, metricType, metricName, metricValue string) error {
	switch metricType {
	case model.Gauge:
		val, err := strconv.ParseFloat(metricValue, 64)
		if err != nil {
			return err
		}
		store.UpdateGauge(metricName, val)
	case model.Counter:
		val, err := strconv.ParseInt(metricValue, 10, 64)
		if err != nil {
			return err
		}
		store.UpdateCounter(metricName, val)
	}
	return nil
}
