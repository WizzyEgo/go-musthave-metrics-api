package service

import (
	"net/http"
	"sync"
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
	key            string
	pollInterval   time.Duration
	reportInterval time.Duration
	rateLimit      int
	reportMu       sync.Mutex
}

func New(cfg config.AgentConfig, log *zap.Logger) *Service {
	if log == nil {
		log = zap.NewNop()
	}
	return &Service{
		metrics: model.NewAgent(),
		client: &http.Client{
			Transport: newRetryTransport(http.DefaultTransport, log),
		},
		log:            log,
		serverURL:      "http://" + cfg.Address,
		key:            cfg.Key,
		pollInterval:   cfg.PollInterval,
		reportInterval: cfg.ReportInterval,
		rateLimit:      cfg.RateLimit,
	}
}

func (s *Service) workerCount() int {
	if s.rateLimit < 1 {
		return 1
	}
	return s.rateLimit
}
