package database

import (
	"context"
	"errors"
	"testing"
)

func TestOpenEmptyDSN(t *testing.T) {
	client, err := Open("")
	if !errors.Is(err, ErrNotConfigured) {
		t.Fatalf("Open() error = %v, want %v", err, ErrNotConfigured)
	}
	if client != nil {
		t.Fatal("Open() client != nil")
	}
}

func TestNilClientPingAndClose(t *testing.T) {
	var client *Client
	if err := client.Ping(context.Background()); !errors.Is(err, ErrNotConfigured) {
		t.Fatalf("Ping() error = %v, want %v", err, ErrNotConfigured)
	}
	if err := client.Close(); err != nil {
		t.Fatalf("Close() error = %v", err)
	}
}
