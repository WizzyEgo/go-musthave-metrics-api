package service

import (
	"net/http"
	"time"

	"go-musthave-metrics-tpl/internal/agent/config"
	"go-musthave-metrics-tpl/internal/agent/model"
)

type Service struct {
	metrics        *model.Agent
	client         *http.Client
	serverURL      string
	pollInterval   time.Duration
	reportInterval time.Duration
}

func New(cfg config.AgentConfig) *Service {
	return &Service{
		metrics:        model.NewAgent(),
		client:         &http.Client{Timeout: 5 * time.Second},
		serverURL:      "http://" + cfg.Address,
		pollInterval:   cfg.PollInterval,
		reportInterval: cfg.ReportInterval,
	}
}

func (s *Service) GetMetrics() *model.Agent {
	return s.metrics
}

func (s *Service) SetServerURL(url string) {
	s.serverURL = url
}
