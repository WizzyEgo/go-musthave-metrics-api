package config

import (
	"errors"
	"flag"
	"io"
	"net"
	"os"
	"reflect"
	"strconv"
	"time"

	"github.com/caarlos0/env/v6"
	"go.uber.org/zap"

	"go-musthave-metrics-tpl/internal/server/logger"
)

const (
	// DefaultAddress — адрес HTTP-сервера по умолчанию.
	DefaultAddress = "localhost:8080"
	// DefaultLogLevel — уровень логирования по умолчанию.
	DefaultLogLevel = "debug"
	// DefaultStoreInterval — период сохранения метрик на диск.
	DefaultStoreInterval = 300 * time.Second
	// DefaultFileStoragePath — файл с текущими значениями метрик.
	// Пустое значение отключает запись на диск.
	DefaultFileStoragePath = "/tmp/metrics-db.json"
	// DefaultRestore — загружать ли файл метрик при старте сервера.
	DefaultRestore = true
)

type ServerConfig struct {
	Address         string        `env:"ADDRESS"`
	LogLevel        string        `env:"LOG_LEVEL"`
	StoreInterval   time.Duration `env:"STORE_INTERVAL"`
	FileStoragePath string        `env:"FILE_STORAGE_PATH"`
	Restore         bool          `env:"RESTORE"`
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
		Address:         DefaultAddress,
		LogLevel:        DefaultLogLevel,
		StoreInterval:   DefaultStoreInterval,
		FileStoragePath: DefaultFileStoragePath,
		Restore:         DefaultRestore,
	}
	storeSec := int(DefaultStoreInterval / time.Second)

	fs := flag.NewFlagSet("server", flag.ContinueOnError)
	fs.SetOutput(output)
	fs.StringVar(&cfg.Address, "a", DefaultAddress, "адрес эндпоинта HTTP-сервера")
	fs.StringVar(&cfg.LogLevel, "l", DefaultLogLevel, "уровень логирования")
	fs.IntVar(&storeSec, "i", int(DefaultStoreInterval/time.Second), "интервал сохранения метрик на диск в секундах (0 — синхронная запись)")
	fs.StringVar(&cfg.FileStoragePath, "f", DefaultFileStoragePath, "файл для сохранения метрик (пустое значение отключает запись)")
	fs.BoolVar(&cfg.Restore, "r", DefaultRestore, "загружать сохранённые метрики при старте сервера")
	if err := fs.Parse(args); err != nil {
		return ServerConfig{}, err
	}

	cfg.StoreInterval = time.Duration(storeSec) * time.Second

	if err := env.ParseWithFuncs(&cfg, secondsParsers(), env.Options{Environment: environ}); err != nil {
		return ServerConfig{}, err
	}
	applyFileStoragePath(&cfg, environ)
	return cfg, nil
}

func applyFileStoragePath(cfg *ServerConfig, environ map[string]string) {
	const key = "FILE_STORAGE_PATH"
	if environ != nil {
		if value, ok := environ[key]; ok {
			cfg.FileStoragePath = value
		}
		return
	}
	if value, ok := os.LookupEnv(key); ok {
		cfg.FileStoragePath = value
	}
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
		zap.Duration("store_interval", cfg.StoreInterval),
		zap.String("file_storage_path", cfg.FileStoragePath),
		zap.Bool("restore", cfg.Restore),
	)
}
