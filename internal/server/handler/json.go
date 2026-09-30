package handler

import (
	"net/http"

	"github.com/mailru/easyjson"

	"go-musthave-metrics-tpl/internal/server/model"
	"go-musthave-metrics-tpl/internal/server/storage"
)

func UpdateJSON(store storage.Storage) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var metric model.Metrics
		if err := easyjson.UnmarshalFromReader(r.Body, &metric); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		if metric.ID == "" {
			w.WriteHeader(http.StatusNotFound)
			return
		}

		switch metric.MType {
		case model.Gauge:
			if metric.Value == nil {
				w.WriteHeader(http.StatusBadRequest)
				return
			}
			store.UpdateGauge(metric.ID, *metric.Value)
			metric.Delta = nil
		case model.Counter:
			if metric.Delta == nil {
				w.WriteHeader(http.StatusBadRequest)
				return
			}
			store.UpdateCounter(metric.ID, *metric.Delta)
			value, ok := store.GetCounter(metric.ID)
			if !ok {
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
			metric.Delta = &value
			metric.Value = nil
		default:
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		writeJSON(w, metric)
	}
}

func ValueJSON(store storage.Storage) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var metric model.Metrics
		if err := easyjson.UnmarshalFromReader(r.Body, &metric); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		switch metric.MType {
		case model.Gauge:
			value, ok := store.GetGauge(metric.ID)
			if !ok {
				w.WriteHeader(http.StatusNotFound)
				return
			}
			metric.Value = &value
			metric.Delta = nil
		case model.Counter:
			value, ok := store.GetCounter(metric.ID)
			if !ok {
				w.WriteHeader(http.StatusNotFound)
				return
			}
			metric.Delta = &value
			metric.Value = nil
		default:
			w.WriteHeader(http.StatusNotFound)
			return
		}

		writeJSON(w, metric)
	}
}

func writeJSON(w http.ResponseWriter, v easyjson.Marshaler) {
	payload, err := easyjson.Marshal(v)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(payload)
}
