package retry

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestDoRetriesOnlyRetriableErrors(t *testing.T) {
	backoff := []time.Duration{time.Millisecond, time.Millisecond, time.Millisecond}

	t.Run("success", func(t *testing.T) {
		calls := 0
		err := do(context.Background(), backoff, func() error {
			calls++
			return nil
		})
		if err != nil {
			t.Fatalf("Do() error = %v", err)
		}
		if calls != 1 {
			t.Fatalf("calls = %d, want 1", calls)
		}
	})

	t.Run("permanent", func(t *testing.T) {
		calls := 0
		permanent := errors.New("permanent")
		err := do(context.Background(), backoff, func() error {
			calls++
			return permanent
		})
		if !errors.Is(err, permanent) {
			t.Fatalf("Do() error = %v", err)
		}
		if calls != 1 {
			t.Fatalf("calls = %d, want 1", calls)
		}
	})

	t.Run("three extra attempts", func(t *testing.T) {
		calls := 0
		temporary := errors.New("temporary")
		err := do(context.Background(), backoff, func() error {
			calls++
			return Retriable(temporary)
		})
		if !errors.Is(err, temporary) {
			t.Fatalf("Do() error = %v", err)
		}
		if calls != len(backoff)+1 {
			t.Fatalf("calls = %d, want %d", calls, len(backoff)+1)
		}
	})

	t.Run("recovers", func(t *testing.T) {
		calls := 0
		err := do(context.Background(), backoff, func() error {
			calls++
			if calls < 3 {
				return Retriable(errors.New("temporary"))
			}
			return nil
		})
		if err != nil {
			t.Fatalf("Do() error = %v", err)
		}
		if calls != 3 {
			t.Fatalf("calls = %d, want 3", calls)
		}
	})
}

func TestDoStopsWhenContextCanceled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	calls := 0
	err := do(ctx, []time.Duration{time.Hour, time.Hour, time.Hour}, func() error {
		calls++
		cancel()
		return Retriable(errors.New("temporary"))
	})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("Do() error = %v, want context.Canceled", err)
	}
	if calls != 1 {
		t.Fatalf("calls = %d, want 1", calls)
	}
}

func TestDefaultBackoff(t *testing.T) {
	want := []time.Duration{time.Second, 3 * time.Second, 5 * time.Second}
	got := backoffDelays()
	if len(got) != len(want) {
		t.Fatalf("backoff len = %d, want %d", len(got), len(want))
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("backoff[%d] = %s, want %s", i, got[i], want[i])
		}
	}
}
