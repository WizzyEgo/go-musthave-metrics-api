package logger

import "testing"

func TestNew(t *testing.T) {
	if _, err := New("no-such-level"); err == nil {
		t.Fatal("expected error for unknown level")
	}
	log, err := New("info")
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	if log == nil {
		t.Fatal("New() returned nil logger")
	}
}
