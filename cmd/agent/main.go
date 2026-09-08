package main

import (
	"go-musthave-metrics-tpl/internal/agent"
	"go-musthave-metrics-tpl/internal/config"
)

func main() {
	cfg := config.ParseAgent()
	agent.New(cfg).Run()
}
