package storage

import (
	"strings"
	"testing"

	"go-musthave-metrics-tpl/migrations"
)

func TestMetricsMigrationUsesDoublePrecision(t *testing.T) {
	data, err := migrations.FS.ReadFile("00001_create_metrics.sql")
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	sql := string(data)
	if !strings.Contains(strings.ToLower(sql), "double precision") {
		t.Fatalf("migration does not use double precision:\n%s", sql)
	}
	if !strings.Contains(sql, "-- +goose Up") {
		t.Fatal("migration is not a goose script")
	}
}
