package config

import (
	"errors"
	"flag"
	"io"
	"os"

	"github.com/caarlos0/env/v6"
)

const (
	// DefaultAddress — адрес HTTP-сервера по умолчанию.
	DefaultAddress = "localhost:8080"
)

type ServerConfig struct {
	Address string `env:"ADDRESS"`
}

func ParseServer() ServerConfig {
	cfg, err := parseServer(os.Args[1:], os.Stderr, nil)
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
	return parseServer(args, output, map[string]string{})
}

func parseServer(args []string, output io.Writer, environ map[string]string) (ServerConfig, error) {
	cfg := ServerConfig{Address: DefaultAddress}
	fs := flag.NewFlagSet("server", flag.ContinueOnError)
	fs.SetOutput(output)
	fs.StringVar(&cfg.Address, "a", DefaultAddress, "адрес эндпоинта HTTP-сервера")
	if err := fs.Parse(args); err != nil {
		return ServerConfig{}, err
	}
	if err := env.Parse(&cfg, env.Options{Environment: environ}); err != nil {
		return ServerConfig{}, err
	}
	return cfg, nil
}
