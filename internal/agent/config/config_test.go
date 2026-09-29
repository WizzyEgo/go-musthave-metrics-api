package config

import (
	"errors"
	"flag"
	"io"
	"strings"
	"testing"
	"time"
)

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
	if cfg.ReportInterval != 4*time.Second {
		t.Fatalf("ReportInterval = %s, want 4s", cfg.ReportInterval)
	}
	if cfg.PollInterval != 1*time.Second {
		t.Fatalf("PollInterval = %s, want 1s", cfg.PollInterval)
	}
}

func TestParseAgentEnvOverridesFlags(t *testing.T) {
	cfg, err := parseAgent([]string{"-a=localhost:1111", "-r=4", "-p=1"}, io.Discard, map[string]string{
		"ADDRESS":         "localhost:2222",
		"REPORT_INTERVAL": "8",
		"POLL_INTERVAL":   "3",
	})
	if err != nil {
		t.Fatalf("parseAgent() error = %v", err)
	}
	if cfg.Address != "localhost:2222" {
		t.Fatalf("Address = %q, want localhost:2222", cfg.Address)
	}
	if cfg.ReportInterval != 8*time.Second {
		t.Fatalf("ReportInterval = %s, want 8s", cfg.ReportInterval)
	}
	if cfg.PollInterval != 3*time.Second {
		t.Fatalf("PollInterval = %s, want 3s", cfg.PollInterval)
	}
}

func TestParseAgentEnvPartialOverride(t *testing.T) {
	cfg, err := parseAgent([]string{"-a=localhost:1111", "-r=4", "-p=1"}, io.Discard, map[string]string{
		"ADDRESS": "127.0.0.1:7777",
	})
	if err != nil {
		t.Fatalf("parseAgent() error = %v", err)
	}
	if cfg.Address != "127.0.0.1:7777" {
		t.Fatalf("Address = %q, want 127.0.0.1:7777", cfg.Address)
	}
	if cfg.ReportInterval != 4*time.Second {
		t.Fatalf("ReportInterval = %s, want 4s", cfg.ReportInterval)
	}
	if cfg.PollInterval != 1*time.Second {
		t.Fatalf("PollInterval = %s, want 1s", cfg.PollInterval)
	}
}

func TestParseAgentEnvInvalidInterval(t *testing.T) {
	_, err := parseAgent(nil, io.Discard, map[string]string{
		"POLL_INTERVAL": "abc",
	})
	if err == nil {
		t.Fatal("expected error for invalid POLL_INTERVAL")
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
