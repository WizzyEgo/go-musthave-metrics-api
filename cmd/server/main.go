package main

import (
	"context"
	"errors"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"go.uber.org/zap"

	"go-musthave-metrics-tpl/internal/server/config"
	"go-musthave-metrics-tpl/internal/server/handler"
	"go-musthave-metrics-tpl/internal/server/logger"
	"go-musthave-metrics-tpl/internal/server/storage"
)

func main() {
	cfg := config.ParseServer()
	defer func() { _ = logger.Log.Sync() }()

	store := storage.NewFileStorage(cfg.FileStoragePath, cfg.StoreInterval == 0)
	if cfg.Restore {
		if err := store.Load(); err != nil {
			logger.Log.Error("failed to restore metrics",
				zap.String("path", cfg.FileStoragePath),
				zap.Error(err),
			)
		}
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	if cfg.FileStoragePath != "" && cfg.StoreInterval > 0 {
		go store.FlushEvery(ctx, cfg.StoreInterval)
	}

	srv := &http.Server{
		Addr:    cfg.Address,
		Handler: handler.MetricHandler(store),
	}

	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := srv.Shutdown(shutdownCtx); err != nil {
			logger.Log.Error("failed to shutdown server", zap.Error(err))
		}
	}()

	if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		logger.Log.Error("failed to start server", zap.Error(err))
		panic(err)
	}

	if err := store.Save(); err != nil {
		logger.Log.Error("failed to save metrics", zap.Error(err))
	}
}
