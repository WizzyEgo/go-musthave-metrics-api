package handler

import (
	"net/http"

	"github.com/go-chi/chi/v5"

	"go-musthave-metrics-tpl/internal/storage"
)

func MetricHandler() http.Handler {
	return NewRouter(storage.NewMemStorage())
}

func NewRouter(store storage.Storage) http.Handler {
	r := chi.NewRouter()
	r.Post("/update/{type}/{name}/{value}", Update(store))
	r.Get("/value/{type}/{name}", Value(store))
	r.Get("/", Index(store))
	return r
}
