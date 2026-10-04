package config

import (
	"io"
	"strings"
	"testing"
	"time"
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
	if cfg.StoreInterval != DefaultStoreInterval {
		t.Fatalf("StoreInterval = %s, want %s", cfg.StoreInterval, DefaultStoreInterval)
	}
	if cfg.FileStoragePath != DefaultFileStoragePath {
		t.Fatalf("FileStoragePath = %q, want %q", cfg.FileStoragePath, DefaultFileStoragePath)
	}
	if cfg.Restore != DefaultRestore {
		t.Fatalf("Restore = %v, want %v", cfg.Restore, DefaultRestore)
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

func TestParseServerStorageFlags(t *testing.T) {
	cfg, err := parseServerFlags([]string{"-i=0", "-f=metrics.json", "-r=false"}, io.Discard)
	if err != nil {
		t.Fatalf("parseServerFlags() error = %v", err)
	}
	if cfg.StoreInterval != 0 {
		t.Fatalf("StoreInterval = %s, want 0", cfg.StoreInterval)
	}
	if cfg.FileStoragePath != "metrics.json" {
		t.Fatalf("FileStoragePath = %q, want metrics.json", cfg.FileStoragePath)
	}
	if cfg.Restore {
		t.Fatal("Restore = true, want false")
	}
}

func TestParseServerStorageEnvOverridesFlags(t *testing.T) {
	cfg, err := parseServer([]string{"-i=10", "-f=flag.json", "-r=true"}, io.Discard, map[string]string{
		"STORE_INTERVAL":    "2",
		"FILE_STORAGE_PATH": "env.json",
		"RESTORE":           "false",
	})
	if err != nil {
		t.Fatalf("parseServer() error = %v", err)
	}
	if cfg.StoreInterval != 2*time.Second {
		t.Fatalf("StoreInterval = %s, want 2s", cfg.StoreInterval)
	}
	if cfg.FileStoragePath != "env.json" {
		t.Fatalf("FileStoragePath = %q, want env.json", cfg.FileStoragePath)
	}
	if cfg.Restore {
		t.Fatal("Restore = true, want false")
	}
}

func TestParseServerStorageEnvOnly(t *testing.T) {
	cfg, err := parseServer(nil, io.Discard, map[string]string{
		"STORE_INTERVAL":    "0",
		"FILE_STORAGE_PATH": "only-env.json",
		"RESTORE":           "true",
	})
	if err != nil {
		t.Fatalf("parseServer() error = %v", err)
	}
	if cfg.StoreInterval != 0 {
		t.Fatalf("StoreInterval = %s, want 0", cfg.StoreInterval)
	}
	if cfg.FileStoragePath != "only-env.json" {
		t.Fatalf("FileStoragePath = %q, want only-env.json", cfg.FileStoragePath)
	}
	if !cfg.Restore {
		t.Fatal("Restore = false, want true")
	}
}

func TestParseServerEmptyFileStoragePathDisablesStore(t *testing.T) {
	cfg, err := parseServer([]string{"-f=flag.json"}, io.Discard, map[string]string{
		"FILE_STORAGE_PATH": "",
	})
	if err != nil {
		t.Fatalf("parseServer() error = %v", err)
	}
	if cfg.FileStoragePath != "" {
		t.Fatalf("FileStoragePath = %q, want empty", cfg.FileStoragePath)
	}
	if cfg.StoreInterval != DefaultStoreInterval {
		t.Fatalf("StoreInterval = %s, want %s", cfg.StoreInterval, DefaultStoreInterval)
	}
}

func TestParseServerStorageEnvPartialOverride(t *testing.T) {
	cfg, err := parseServer([]string{"-i=4", "-f=flag.json", "-r=false"}, io.Discard, map[string]string{
		"STORE_INTERVAL": "8",
	})
	if err != nil {
		t.Fatalf("parseServer() error = %v", err)
	}
	if cfg.StoreInterval != 8*time.Second {
		t.Fatalf("StoreInterval = %s, want 8s", cfg.StoreInterval)
	}
	if cfg.FileStoragePath != "flag.json" {
		t.Fatalf("FileStoragePath = %q, want flag.json", cfg.FileStoragePath)
	}
	if cfg.Restore {
		t.Fatal("Restore = true, want false")
	}
}

func TestParseServerInvalidStoreInterval(t *testing.T) {
	_, err := parseServer(nil, io.Discard, map[string]string{
		"STORE_INTERVAL": "abc",
	})
	if err == nil {
		t.Fatal("expected error for invalid STORE_INTERVAL")
	}
}

func TestParseServerInvalidRestore(t *testing.T) {
	_, err := parseServer(nil, io.Discard, map[string]string{
		"RESTORE": "maybe",
	})
	if err == nil {
		t.Fatal("expected error for invalid RESTORE")
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
