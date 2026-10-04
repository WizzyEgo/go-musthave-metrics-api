package config

import (
	"errors"
	"flag"
	"io"
	"os"
	"reflect"
	"strconv"
	"time"

	"github.com/caarlos0/env/v11"
)

const (
	// DefaultAddress — адрес HTTP-сервера по умолчанию.
	DefaultAddress = "localhost:8080"
	// DefaultPollInterval — интервал опроса метрик из runtime.
	DefaultPollInterval = 2 * time.Second
	// DefaultReportInterval — интервал отправки метрик на сервер.
	DefaultReportInterval = 10 * time.Second
)

type AgentConfig struct {
	Address        string        `env:"ADDRESS"`
	PollInterval   time.Duration `env:"POLL_INTERVAL"`
	ReportInterval time.Duration `env:"REPORT_INTERVAL"`
}

func DefaultAgent() AgentConfig {
	return AgentConfig{
		Address:        DefaultAddress,
		PollInterval:   DefaultPollInterval,
		ReportInterval: DefaultReportInterval,
	}
}

func ParseAgent() AgentConfig {
	cfg, err := parseAgent(os.Args[1:], os.Stderr, nil)
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

func parseAgentFlags(args []string, output io.Writer) (AgentConfig, error) {
	return parseAgent(args, output, map[string]string{})
}

func parseAgent(args []string, output io.Writer, environ map[string]string) (AgentConfig, error) {
	cfg := DefaultAgent()
	pollSec := int(DefaultPollInterval / time.Second)
	reportSec := int(DefaultReportInterval / time.Second)

	fs := flag.NewFlagSet("agent", flag.ContinueOnError)
	fs.SetOutput(output)
	fs.StringVar(&cfg.Address, "a", DefaultAddress, "адрес эндпоинта HTTP-сервера")
	fs.IntVar(&reportSec, "r", int(DefaultReportInterval/time.Second), "частота отправки метрик на сервер в секундах")
	fs.IntVar(&pollSec, "p", int(DefaultPollInterval/time.Second), "частота опроса метрик из runtime в секундах")
	if err := fs.Parse(args); err != nil {
		return AgentConfig{}, err
	}

	cfg.PollInterval = time.Duration(pollSec) * time.Second
	cfg.ReportInterval = time.Duration(reportSec) * time.Second

	if err := env.ParseWithOptions(&cfg, env.Options{
		Environment: environ,
		FuncMap:     secondsParsers(),
	}); err != nil {
		return AgentConfig{}, err
	}
	return cfg, nil
}

func secondsParsers() map[reflect.Type]env.ParserFunc {
	return map[reflect.Type]env.ParserFunc{
		reflect.TypeOf(time.Duration(0)): func(v string) (interface{}, error) {
			seconds, err := strconv.Atoi(v)
			if err != nil {
				return nil, err
			}
			return time.Duration(seconds) * time.Second, nil
		},
	}
}
