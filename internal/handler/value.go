package handler

import (
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	"go-musthave-metrics-tpl/internal/model"
	"go-musthave-metrics-tpl/internal/model/server"
)

func Value(store server.Storage) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		metricType := chi.URLParam(r, "type")
		metricName := chi.URLParam(r, "name")

		var (
			body string
			ok   bool
		)

		switch metricType {
		case model.Gauge:
			var value float64
			value, ok = store.GetGauge(metricName)
			if ok {
				body = strconv.FormatFloat(value, 'f', -1, 64)
			}
		case model.Counter:
			var value int64
			value, ok = store.GetCounter(metricName)
			if ok {
				body = strconv.FormatInt(value, 10)
			}
		default:
			w.WriteHeader(http.StatusNotFound)
			return
		}

		if !ok {
			w.WriteHeader(http.StatusNotFound)
			return
		}

		w.Header().Set("Content-Type", "text/plain")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(body))
	}
}
