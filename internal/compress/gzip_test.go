package compress

import (
	"bytes"
	"compress/gzip"
	"io"
	"testing"
)

func TestEncodeRoundTrip(t *testing.T) {
	payload := []byte(`{"id":"Alloc","type":"gauge","value":1}`)
	compressed, err := Encode(payload)
	if err != nil {
		t.Fatalf("Encode() error = %v", err)
	}

	zr, err := NewReader(bytes.NewReader(compressed))
	if err != nil {
		t.Fatalf("NewReader() error = %v", err)
	}
	got, err := io.ReadAll(zr)
	if err != nil {
		t.Fatalf("read gzip: %v", err)
	}
	if err := zr.Close(); err != nil {
		t.Fatalf("close gzip: %v", err)
	}
	if !bytes.Equal(got, payload) {
		t.Fatalf("payload = %q, want %q", got, payload)
	}

	if _, err := gzip.NewReader(bytes.NewReader(compressed)); err != nil {
		t.Fatalf("stdlib gzip reader: %v", err)
	}
}
