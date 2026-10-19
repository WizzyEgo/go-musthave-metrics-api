package handler

import (
	"errors"
	"net/http"

	"github.com/mailru/easyjson"
	"go.uber.org/zap"

	"go-musthave-metrics-tpl/internal/model"
	"go-musthave-metrics-tpl/internal/server/storage"
)

func Updates(log *zap.Logger, store storage.Storage) http.HandlerFunc {
	if log == nil {
		log = zap.NewNop()
	}
	return func(w http.ResponseWriter, r *http.Request) {
		var metrics model.MetricsList
		if err := easyjson.UnmarshalFromReader(r.Body, &metrics); err != nil || metrics == nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		updated, err := store.UpdateBatch(r.Context(), metrics)
		if err != nil {
			writeBatchError(log, w, err)
			return
		}
		writeJSON(log, w, model.MetricsList(updated))
	}
}

func writeBatchError(log *zap.Logger, w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, storage.ErrEmptyMetricID):
		w.WriteHeader(http.StatusNotFound)
	case errors.Is(err, storage.ErrInvalidMetric):
		w.WriteHeader(http.StatusBadRequest)
	default:
		writeInternalError(log, w, "failed to update metrics batch", err)
	}
}
