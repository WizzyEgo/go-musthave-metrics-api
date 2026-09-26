package agent

import (
	"net/http"
	"time"

	"go-musthave-metrics-tpl/internal/config"
	modelagent "go-musthave-metrics-tpl/internal/model/agent"
)

type Service struct {
	metrics   *modelagent.Agent
	client    *http.Client
	serverURL string
	pollSec   int
	reportSec int
}

func New() *Service {
	return &Service{
		metrics:   modelagent.New(),
		client:    &http.Client{Timeout: 5 * time.Second},
		serverURL: config.DefaultServerURL,
		pollSec:   config.PollInterval,
		reportSec: config.ReportInterval,
	}
}

func (s *Service) GetMetrics() *modelagent.Agent {
	return s.metrics
}

func (s *Service) SetServerURL(url string) {
	s.serverURL = url
}
