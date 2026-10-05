package main

import (
	"go-musthave-metrics-tpl/internal/agent/config"
	"go-musthave-metrics-tpl/internal/agent/service"
)

func main() {
	cfg := config.ParseAgent()
	service.New(cfg).Run()
}
