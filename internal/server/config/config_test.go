package config

import (
	"io"
	"strings"
	"testing"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest/observer"

	"go-musthave-metrics-tpl/internal/server/logger"
)

func TestParseServerFlagsDefaults(t *testing.T) {
	cfg, err := parseServerFlags(nil, io.Discard)
	if err != nil {
		t.Fatalf("parseServerFlags() error = %v", err)
	}
	if cfg.Address != DefaultAddress {
		t.Fatalf("Address = %q, want %q", cfg.Address, DefaultAddress)
	}
	if cfg.LogLevel != DefaultLogLevel {
		t.Fatalf("LogLevel = %q, want %q", cfg.LogLevel, DefaultLogLevel)
	}
}

func TestParseServerFlagsAddress(t *testing.T) {
	cfg, err := parseServerFlags([]string{"-a=127.0.0.1:9090"}, io.Discard)
	if err != nil {
		t.Fatalf("parseServerFlags() error = %v", err)
	}
	if cfg.Address != "127.0.0.1:9090" {
		t.Fatalf("Address = %q, want 127.0.0.1:9090", cfg.Address)
	}
}

func TestParseServerEnvOverridesFlag(t *testing.T) {
	cfg, err := parseServer([]string{"-a=127.0.0.1:9090"}, io.Discard, map[string]string{
		"ADDRESS": "localhost:7777",
	})
	if err != nil {
		t.Fatalf("parseServer() error = %v", err)
	}
	if cfg.Address != "localhost:7777" {
		t.Fatalf("Address = %q, want localhost:7777", cfg.Address)
	}
}

func TestParseServerEnvOnly(t *testing.T) {
	cfg, err := parseServer(nil, io.Discard, map[string]string{
		"ADDRESS": "127.0.0.1:3333",
	})
	if err != nil {
		t.Fatalf("parseServer() error = %v", err)
	}
	if cfg.Address != "127.0.0.1:3333" {
		t.Fatalf("Address = %q, want 127.0.0.1:3333", cfg.Address)
	}
}

func TestParseServerLogLevelFlag(t *testing.T) {
	cfg, err := parseServerFlags([]string{"-l=warn"}, io.Discard)
	if err != nil {
		t.Fatalf("parseServerFlags() error = %v", err)
	}
	if cfg.LogLevel != "warn" {
		t.Fatalf("LogLevel = %q, want warn", cfg.LogLevel)
	}
}

func TestParseServerLogLevelEnvOverridesFlag(t *testing.T) {
	cfg, err := parseServer([]string{"-l=debug"}, io.Discard, map[string]string{
		"LOG_LEVEL": "error",
	})
	if err != nil {
		t.Fatalf("parseServer() error = %v", err)
	}
	if cfg.LogLevel != "error" {
		t.Fatalf("LogLevel = %q, want error", cfg.LogLevel)
	}
}

func TestParseServerLogLevelEnvOnly(t *testing.T) {
	cfg, err := parseServer(nil, io.Discard, map[string]string{
		"LOG_LEVEL": "info",
	})
	if err != nil {
		t.Fatalf("parseServer() error = %v", err)
	}
	if cfg.LogLevel != "info" {
		t.Fatalf("LogLevel = %q, want info", cfg.LogLevel)
	}
}

func TestLogStartup(t *testing.T) {
	core, recorded := observer.New(zapcore.DebugLevel)
	prev := logger.Log
	logger.Log = zap.New(core)
	t.Cleanup(func() { logger.Log = prev })

	logStartup(ServerConfig{Address: "127.0.0.1:9090"}, []string{"-a=127.0.0.1:9090", "-l=debug"})

	entries := recorded.All()
	if len(entries) != 1 {
		t.Fatalf("log entries = %d, want 1", len(entries))
	}
	fields := entries[0].ContextMap()
	if fields["address"] != "127.0.0.1" {
		t.Fatalf("address = %v, want 127.0.0.1", fields["address"])
	}
	if fields["port"] != "9090" {
		t.Fatalf("port = %v, want 9090", fields["port"])
	}
	flags, ok := fields["flags"].([]any)
	if !ok || len(flags) != 2 || flags[0] != "-a=127.0.0.1:9090" || flags[1] != "-l=debug" {
		t.Fatalf("flags = %v", fields["flags"])
	}
}

func TestParseServerFlagsUnknown(t *testing.T) {
	_, err := parseServerFlags([]string{"-unknown=1"}, io.Discard)
	if err == nil {
		t.Fatal("expected error for unknown flag")
	}
	if !strings.Contains(err.Error(), "flag provided but not defined") {
		t.Fatalf("unexpected error: %v", err)
	}
}
