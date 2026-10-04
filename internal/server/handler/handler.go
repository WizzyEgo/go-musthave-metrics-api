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
	r.Post("/update/{type}/{name}/{value}", Update(store))
	r.Post("/update", UpdateJSON(store))
	r.Post("/update/", UpdateJSON(store))
	r.Get("/value/{type}/{name}", Value(store))
	r.Post("/value", ValueJSON(store))
	r.Post("/value/", ValueJSON(store))
	r.Get("/", Index(store))
	return r
}
