package storage

import (
	"os"
	"testing"

	_ "github.com/jackc/pgx/v5/stdlib"
	"go.uber.org/zap"

	"go-musthave-metrics-tpl/internal/server/database"
)

func TestPostgresStorageRoundTrip(t *testing.T) {
	dsn := os.Getenv("TEST_DATABASE_DSN")
	if dsn == "" {
		t.Skip("TEST_DATABASE_DSN is not set")
	}

	client, err := database.Open(dsn)
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	t.Cleanup(func() { _ = client.Close() })

	store, err := NewPostgresStorage(client.DB(), zap.NewNop())
	if err != nil {
		t.Fatalf("NewPostgresStorage() error = %v", err)
	}

	store.UpdateGauge("Alloc", 1.5)
	store.UpdateGauge("Alloc", 2.25)
	store.UpdateCounter("PollCount", 2)
	store.UpdateCounter("PollCount", 3)

	gauge, ok := store.GetGauge("Alloc")
	if !ok || gauge != 2.25 {
		t.Fatalf("GetGauge() = (%v, %v), want (2.25, true)", gauge, ok)
	}
	counter, ok := store.GetCounter("PollCount")
	if !ok || counter != 5 {
		t.Fatalf("GetCounter() = (%v, %v), want (5, true)", counter, ok)
	}

	gauges, counters := store.GetAll()
	if gauges["Alloc"] != 2.25 {
		t.Fatalf("GetAll gauges Alloc = %v, want 2.25", gauges["Alloc"])
	}
	if counters["PollCount"] != 5 {
		t.Fatalf("GetAll counters PollCount = %v, want 5", counters["PollCount"])
	}
}
