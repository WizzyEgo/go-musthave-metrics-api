package main

import (
	"net/http"

	"go-musthave-metrics-tpl/internal/config"
	"go-musthave-metrics-tpl/internal/handler"
)

func main() {
	cfg := config.ParseServer()
	err := http.ListenAndServe(cfg.Address, handler.MetricHandler())

	if err != nil {
		panic(err)
	}
}
