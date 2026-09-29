package handler

import (
	"net/http"

	"github.com/go-chi/chi/v5"

	"go-musthave-metrics-tpl/internal/server/logger"
	"go-musthave-metrics-tpl/internal/server/storage"
)

func MetricHandler() http.Handler {
	return NewRouter(storage.NewMemStorage())
}

func NewRouter(store storage.Storage) http.Handler {
	r := chi.NewRouter()
	r.Use(logger.RequestLogger)
	r.Post("/update/{type}/{name}/{value}", Update(store))
	r.Post("/update", UpdateJSON(store))
	r.Post("/update/", UpdateJSON(store))
	r.Get("/value/{type}/{name}", Value(store))
	r.Post("/value", ValueJSON(store))
	r.Post("/value/", ValueJSON(store))
	r.Get("/", Index(store))
	return r
}
