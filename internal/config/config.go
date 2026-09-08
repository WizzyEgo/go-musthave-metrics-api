package config

import (
	"errors"
	"flag"
	"io"
	"os"
)

const (
	// DefaultAddress — адрес HTTP-сервера по умолчанию.
	DefaultAddress = "localhost:8080"
	// DefaultPollInterval — интервал опроса метрик из runtime в секундах.
	DefaultPollInterval = 2
	// DefaultReportInterval — интервал отправки метрик на сервер в секундах.
	DefaultReportInterval = 10
)

type ServerConfig struct {
	Address string
}

type AgentConfig struct {
	Address        string
	PollInterval   int
	ReportInterval int
}

func DefaultAgent() AgentConfig {
	return AgentConfig{
		Address:        DefaultAddress,
		PollInterval:   DefaultPollInterval,
		ReportInterval: DefaultReportInterval,
	}
}

func ParseServer() ServerConfig {
	cfg, err := parseServerFlags(os.Args[1:], os.Stderr)
	exitOnFlagError(err)
	return cfg
}

func ParseAgent() AgentConfig {
	cfg, err := parseAgentFlags(os.Args[1:], os.Stderr)
	exitOnFlagError(err)
	return cfg
}

func exitOnFlagError(err error) {
	if err == nil {
		return
	}
	if errors.Is(err, flag.ErrHelp) {
		os.Exit(0)
	}
	os.Exit(2)
}

func parseServerFlags(args []string, output io.Writer) (ServerConfig, error) {
	cfg := ServerConfig{Address: DefaultAddress}
	fs := flag.NewFlagSet("server", flag.ContinueOnError)
	fs.SetOutput(output)
	fs.StringVar(&cfg.Address, "a", DefaultAddress, "адрес эндпоинта HTTP-сервера")
	if err := fs.Parse(args); err != nil {
		return ServerConfig{}, err
	}
	return cfg, nil
}

func parseAgentFlags(args []string, output io.Writer) (AgentConfig, error) {
	cfg := DefaultAgent()
	fs := flag.NewFlagSet("agent", flag.ContinueOnError)
	fs.SetOutput(output)
	fs.StringVar(&cfg.Address, "a", DefaultAddress, "адрес эндпоинта HTTP-сервера")
	fs.IntVar(&cfg.ReportInterval, "r", DefaultReportInterval, "частота отправки метрик на сервер в секундах")
	fs.IntVar(&cfg.PollInterval, "p", DefaultPollInterval, "частота опроса метрик из runtime в секундах")
	if err := fs.Parse(args); err != nil {
		return AgentConfig{}, err
	}
	return cfg, nil
}
