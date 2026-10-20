package handler

import (
	"errors"
	"net/http"

	"github.com/mailru/easyjson"

	"go-musthave-metrics-tpl/internal/model"
	"go-musthave-metrics-tpl/internal/server/storage"
)

func Updates(store storage.Storage) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var metrics model.MetricsList
		if err := easyjson.UnmarshalFromReader(r.Body, &metrics); err != nil || metrics == nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		updated, err := store.UpdateBatch(metrics)
		if err != nil {
			writeBatchError(w, err)
			return
		}
		writeJSON(w, model.MetricsList(updated))
	}
}

func writeBatchError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, storage.ErrEmptyMetricID):
		w.WriteHeader(http.StatusNotFound)
	case errors.Is(err, storage.ErrInvalidMetric):
		w.WriteHeader(http.StatusBadRequest)
	default:
		w.WriteHeader(http.StatusInternalServerError)
	}
}
