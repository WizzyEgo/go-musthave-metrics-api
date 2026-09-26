package main

import (
	"net/http"

	"go-musthave-metrics-tpl/internal/server/config"
	"go-musthave-metrics-tpl/internal/server/handler"
)

func main() {
	cfg := config.ParseServer()
	err := http.ListenAndServe(cfg.Address, handler.MetricHandler())

	if err != nil {
		panic(err)
	}
}
