package config

import (
	"io"
	"strings"
	"testing"
)

func TestParseServerFlagsDefaults(t *testing.T) {
	cfg, err := parseServerFlags(nil, io.Discard)
	if err != nil {
		t.Fatalf("parseServerFlags() error = %v", err)
	}
	if cfg.Address != DefaultAddress {
		t.Fatalf("Address = %q, want %q", cfg.Address, DefaultAddress)
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

func TestParseServerFlagsUnknown(t *testing.T) {
	_, err := parseServerFlags([]string{"-unknown=1"}, io.Discard)
	if err == nil {
		t.Fatal("expected error for unknown flag")
	}
	if !strings.Contains(err.Error(), "flag provided but not defined") {
		t.Fatalf("unexpected error: %v", err)
	}
}
