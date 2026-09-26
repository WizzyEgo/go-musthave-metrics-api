package agent

import (
	"net/http"
	"time"

	"go-musthave-metrics-tpl/internal/config"
	"go-musthave-metrics-tpl/internal/model"
)

type Service struct {
	metrics   *model.Agent
	client    *http.Client
	serverURL string
	pollSec   int
	reportSec int
}

func New(cfg config.AgentConfig) *Service {
	return &Service{
		metrics:   model.NewAgent(),
		client:    &http.Client{Timeout: 5 * time.Second},
		serverURL: "http://" + cfg.Address,
		pollSec:   cfg.PollInterval,
		reportSec: cfg.ReportInterval,
	}
}

func (s *Service) GetMetrics() *model.Agent {
	return s.metrics
}

func (s *Service) SetServerURL(url string) {
	s.serverURL = url
}
