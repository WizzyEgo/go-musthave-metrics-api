package config

import (
	"errors"
	"flag"
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

func TestParseServerFlagsUnknown(t *testing.T) {
	_, err := parseServerFlags([]string{"-unknown=1"}, io.Discard)
	if err == nil {
		t.Fatal("expected error for unknown flag")
	}
	if !strings.Contains(err.Error(), "flag provided but not defined") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestParseAgentFlagsDefaults(t *testing.T) {
	cfg, err := parseAgentFlags(nil, io.Discard)
	if err != nil {
		t.Fatalf("parseAgentFlags() error = %v", err)
	}
	if cfg.Address != DefaultAddress {
		t.Fatalf("Address = %q, want %q", cfg.Address, DefaultAddress)
	}
	if cfg.ReportInterval != DefaultReportInterval {
		t.Fatalf("ReportInterval = %d, want %d", cfg.ReportInterval, DefaultReportInterval)
	}
	if cfg.PollInterval != DefaultPollInterval {
		t.Fatalf("PollInterval = %d, want %d", cfg.PollInterval, DefaultPollInterval)
	}
}

func TestParseAgentFlagsOverrides(t *testing.T) {
	cfg, err := parseAgentFlags([]string{"-a=localhost:9999", "-r=4", "-p=1"}, io.Discard)
	if err != nil {
		t.Fatalf("parseAgentFlags() error = %v", err)
	}
	if cfg.Address != "localhost:9999" {
		t.Fatalf("Address = %q, want localhost:9999", cfg.Address)
	}
	if cfg.ReportInterval != 4 {
		t.Fatalf("ReportInterval = %d, want 4", cfg.ReportInterval)
	}
	if cfg.PollInterval != 1 {
		t.Fatalf("PollInterval = %d, want 1", cfg.PollInterval)
	}
}

func TestParseAgentFlagsUnknown(t *testing.T) {
	_, err := parseAgentFlags([]string{"-z=10"}, io.Discard)
	if err == nil {
		t.Fatal("expected error for unknown flag")
	}
	if !errors.Is(err, flag.ErrHelp) && !strings.Contains(err.Error(), "flag provided but not defined") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestParseAgentFlagsInvalidInterval(t *testing.T) {
	_, err := parseAgentFlags([]string{"-r=abc"}, io.Discard)
	if err == nil {
		t.Fatal("expected error for invalid interval")
	}
}
