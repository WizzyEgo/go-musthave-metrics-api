package main

import (
	"net/http"

	"go.uber.org/zap"

	"go-musthave-metrics-tpl/internal/server/config"
	"go-musthave-metrics-tpl/internal/server/handler"
	"go-musthave-metrics-tpl/internal/server/logger"
)

func main() {
	cfg := config.ParseServer()
	defer func() { _ = logger.Log.Sync() }()

	if err := http.ListenAndServe(cfg.Address, handler.MetricHandler()); err != nil {
		logger.Log.Error("failed to start server", zap.Error(err))
		panic(err)
	}
}
