package config

import "testing"

func TestLoadKeepsDatabaseURLExplicit(t *testing.T) {
	t.Setenv("TRACKMATE__BOT_TOKEN", "test-token")
	t.Setenv("TRACKMATE__DATABASE_URL", "postgres://postgres:postgres@localhost:5432/trackmate?sslmode=disable")
	t.Setenv("TRACKMATE__DEFAULT_TIMEZONE", "Europe/Moscow")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if cfg.DatabaseURL != "postgres://postgres:postgres@localhost:5432/trackmate?sslmode=disable" {
		t.Fatalf("unexpected database url: %q", cfg.DatabaseURL)
	}
}
