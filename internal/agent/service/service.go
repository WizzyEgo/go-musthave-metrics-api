package service

import (
	"net/http"
	"time"

	"go.uber.org/zap"

	"go-musthave-metrics-tpl/internal/agent/config"
	"go-musthave-metrics-tpl/internal/agent/model"
)

type Service struct {
	metrics        *model.Agent
	client         *http.Client
	log            *zap.Logger
	serverURL      string
	pollInterval   time.Duration
	reportInterval time.Duration
}

func New(cfg config.AgentConfig, log *zap.Logger) *Service {
	if log == nil {
		log = zap.NewNop()
	}
	return &Service{
		metrics:        model.NewAgent(),
		client:         &http.Client{Timeout: 5 * time.Second},
		log:            log,
		serverURL:      "http://" + cfg.Address,
		pollInterval:   cfg.PollInterval,
		reportInterval: cfg.ReportInterval,
	}
}
