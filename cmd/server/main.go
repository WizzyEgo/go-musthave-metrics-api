package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
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
	os.Exit(run())
}

func run() int {
	cfg, err := config.ParseServer()
	if err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return 0
		}
		fmt.Fprintf(os.Stderr, "parse config: %v\n", err)
		return 2
	}

	log, err := logger.New(cfg.LogLevel)
	if err != nil {
		fmt.Fprintf(os.Stderr, "init logger: %v\n", err)
		return 1
	}
	defer func() { _ = log.Sync() }()

	log.Info("server starting", zap.String("address", cfg.Address))

	store, err := storage.NewFileStorage(log.Named("storage"), cfg.FileStoragePath, cfg.StoreInterval == 0, cfg.Restore)
	if err != nil {
		log.Error("failed to restore metrics", zap.Error(err))
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	if cfg.FileStoragePath != "" && cfg.StoreInterval > 0 {
		go store.FlushEvery(ctx, cfg.StoreInterval)
	}

	srv := &http.Server{
		Addr:    cfg.Address,
		Handler: handler.MetricHandler(log.Named("http"), store),
	}

	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := srv.Shutdown(shutdownCtx); err != nil {
			log.Error("failed to shutdown server", zap.Error(err))
		}
	}()

	if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Error("failed to start server", zap.Error(err))
		return 1
	}

	if err := store.Save(); err != nil {
		log.Error("failed to save metrics", zap.Error(err))
	}
	return 0
}
