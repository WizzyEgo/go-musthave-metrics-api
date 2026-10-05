package config

import (
	"errors"
	"flag"
	"io"
	"net"
	"os"

	"github.com/caarlos0/env/v6"
	"go.uber.org/zap"

	"go-musthave-metrics-tpl/internal/server/logger"
)

const (
	// DefaultAddress — адрес HTTP-сервера по умолчанию.
	DefaultAddress = "localhost:8080"
	// DefaultLogLevel — уровень логирования по умолчанию.
	DefaultLogLevel = "debug"
)

type ServerConfig struct {
	Address  string `env:"ADDRESS"`
	LogLevel string `env:"LOG_LEVEL"`
}

func ParseServer() ServerConfig {
	flags := os.Args[1:]
	cfg, err := parseServer(flags, os.Stderr, nil)
	exitOnFlagError(err)

	if err := logger.Initialize(cfg.LogLevel); err != nil {
		panic(err)
	}
	logStartup(cfg, flags)
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
	return parseServer(args, output, map[string]string{})
}

func parseServer(args []string, output io.Writer, environ map[string]string) (ServerConfig, error) {
	cfg := ServerConfig{
		Address:  DefaultAddress,
		LogLevel: DefaultLogLevel,
	}
	fs := flag.NewFlagSet("server", flag.ContinueOnError)
	fs.SetOutput(output)
	fs.StringVar(&cfg.Address, "a", DefaultAddress, "адрес эндпоинта HTTP-сервера")
	fs.StringVar(&cfg.LogLevel, "l", DefaultLogLevel, "уровень логирования")
	if err := fs.Parse(args); err != nil {
		return ServerConfig{}, err
	}
	if err := env.Parse(&cfg, env.Options{Environment: environ}); err != nil {
		return ServerConfig{}, err
	}
	return cfg, nil
}

func logStartup(cfg ServerConfig, flags []string) {
	host, port, err := net.SplitHostPort(cfg.Address)
	if err != nil {
		host = cfg.Address
		port = ""
	}

	logger.Log.Debug("server starting",
		zap.String("address", host),
		zap.String("port", port),
		zap.Strings("flags", flags),
	)
}
