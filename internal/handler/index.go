package handler

import (
	"encoding/json"
	"net/http"

	"go-musthave-metrics-tpl/internal/model/server"
)

func Index(store server.Storage) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		gauges, counters := store.GetAll()
		metrics := make(map[string]any, len(gauges)+len(counters))

		for name, value := range gauges {
			metrics[name] = value
		}
		for name, value := range counters {
			metrics[name] = value
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(metrics)
	}
}
