package handler

import (
	"net/http"

	"github.com/mailru/easyjson"
	"go.uber.org/zap"

	"go-musthave-metrics-tpl/internal/model"
	"go-musthave-metrics-tpl/internal/server/storage"
)

func UpdateJSON(log *zap.Logger, store storage.Storage) http.HandlerFunc {
	if log == nil {
		log = zap.NewNop()
	}
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
			if err := store.UpdateGauge(r.Context(), metric.ID, *metric.Value); err != nil {
				writeInternalError(log, w, "failed to update gauge", err)
				return
			}
			metric.Delta = nil
		case model.Counter:
			if metric.Delta == nil {
				w.WriteHeader(http.StatusBadRequest)
				return
			}
			if err := store.UpdateCounter(r.Context(), metric.ID, *metric.Delta); err != nil {
				writeInternalError(log, w, "failed to update counter", err)
				return
			}
			value, ok := store.GetCounter(r.Context(), metric.ID)
			if !ok {
				writeInternalError(log, w, "failed to read counter after update", nil)
				return
			}
			metric.Delta = &value
			metric.Value = nil
		default:
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		writeJSON(log, w, metric)
	}
}

func ValueJSON(log *zap.Logger, store storage.Storage) http.HandlerFunc {
	if log == nil {
		log = zap.NewNop()
	}
	return func(w http.ResponseWriter, r *http.Request) {
		var metric model.Metrics
		if err := easyjson.UnmarshalFromReader(r.Body, &metric); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		switch metric.MType {
		case model.Gauge:
			value, ok := store.GetGauge(r.Context(), metric.ID)
			if !ok {
				w.WriteHeader(http.StatusNotFound)
				return
			}
			metric.Value = &value
			metric.Delta = nil
		case model.Counter:
			value, ok := store.GetCounter(r.Context(), metric.ID)
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

		writeJSON(log, w, metric)
	}
}

func writeJSON(log *zap.Logger, w http.ResponseWriter, v easyjson.Marshaler) {
	payload, err := easyjson.Marshal(v)
	if err != nil {
		writeInternalError(log, w, "failed to encode response", err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(payload)
}
