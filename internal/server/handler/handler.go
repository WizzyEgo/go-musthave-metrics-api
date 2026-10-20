package handler

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"go.uber.org/zap"

	"go-musthave-metrics-tpl/internal/server/middleware"
	"go-musthave-metrics-tpl/internal/server/storage"
)

func MetricHandler(log *zap.Logger, store storage.Storage, pinger ...Pinger) http.Handler {
	return NewRouter(log, store, pinger...)
}

func NewRouter(log *zap.Logger, store storage.Storage, pinger ...Pinger) http.Handler {
	if log == nil {
		log = zap.NewNop()
	}

	var db Pinger
	if len(pinger) > 0 {
		db = pinger[0]
	}

	r := chi.NewRouter()
	r.Use(middleware.RequestLogger(log))
	r.Use(middleware.Gzip)
	r.Get("/ping", Ping(db))
	r.Post("/update/{type}/{name}/{value}", Update(store))
	r.Post("/update", UpdateJSON(store))
	r.Post("/update/", UpdateJSON(store))
	r.Post("/updates", Updates(store))
	r.Post("/updates/", Updates(store))
	r.Get("/value/{type}/{name}", Value(store))
	r.Post("/value", ValueJSON(store))
	r.Post("/value/", ValueJSON(store))
	r.Get("/", Index(store))
	return r
}
