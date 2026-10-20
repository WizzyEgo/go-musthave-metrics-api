package handler

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"go.uber.org/zap"

	"go-musthave-metrics-tpl/internal/server/middleware"
	"go-musthave-metrics-tpl/internal/server/storage"
)

func MetricHandler(log *zap.Logger, store storage.Storage) http.Handler {
	return NewRouter(log, store)
}

func NewRouter(log *zap.Logger, store storage.Storage) http.Handler {
	if log == nil {
		log = zap.NewNop()
	}

	r := chi.NewRouter()
	r.Use(middleware.RequestLogger(log))
	r.Use(middleware.Gzip)
	r.Get("/ping", Ping(log, store))
	r.Post("/update/{type}/{name}/{value}", Update(log, store))
	r.Post("/update", UpdateJSON(log, store))
	r.Post("/update/", UpdateJSON(log, store))
	r.Post("/updates", Updates(log, store))
	r.Post("/updates/", Updates(log, store))
	r.Get("/value/{type}/{name}", Value(store))
	r.Post("/value", ValueJSON(log, store))
	r.Post("/value/", ValueJSON(log, store))
	r.Get("/", Index(store))
	return r
}

func writeInternalError(log *zap.Logger, w http.ResponseWriter, msg string, err error) {
	if log == nil {
		log = zap.NewNop()
	}
	if err != nil {
		log.Error(msg, zap.Error(err))
	} else {
		log.Error(msg)
	}
	w.WriteHeader(http.StatusInternalServerError)
}
