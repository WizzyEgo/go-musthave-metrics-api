package hash

import (
	"crypto/sha256"
	"encoding/base64"
	"testing"
)

func TestSumDependsOnBodyAndKey(t *testing.T) {
	body := []byte(`{"id":"Alloc","type":"gauge","value":1}`)
	first := Sum(body, "secret")
	if first == "" || len(first) != base64.StdEncoding.EncodedLen(sha256.Size) {
		t.Fatalf("sum = %q, want %d base64 chars", first, base64.StdEncoding.EncodedLen(sha256.Size))
	}
	if Sum(body, "secret") != first {
		t.Fatal("sum is not stable")
	}
	if Sum(body, "other") == first {
		t.Fatal("different keys produced the same sum")
	}
	if Sum([]byte(`{"id":"Alloc","type":"gauge","value":2}`), "secret") == first {
		t.Fatal("different bodies produced the same sum")
	}
}

func TestValid(t *testing.T) {
	body := []byte("payload")
	key := "secret"
	sum := Sum(body, key)

	if !Valid(body, key, sum) {
		t.Fatal("expected valid signature")
	}
	if Valid(body, "other", sum) {
		t.Fatal("signature for another key was accepted")
	}
	if Valid([]byte("other"), key, sum) {
		t.Fatal("signature for another body was accepted")
	}
	if Valid(body, key, "zz") {
		t.Fatal("invalid signature was accepted")
	}
	if Valid(body, key, "") {
		t.Fatal("empty signature was accepted")
	}
}
