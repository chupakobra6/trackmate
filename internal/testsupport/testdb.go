package testsupport

import (
	"context"
	"database/sql"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/igor/trackmate/internal/logging"
	"github.com/igor/trackmate/internal/storage/postgres"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"
)

func OpenMigratedStore(t *testing.T) (*postgres.Store, string) {
	t.Helper()
	baseURL := os.Getenv("TRACKMATE_TEST_DATABASE_URL")
	if baseURL == "" {
		t.Skip("TRACKMATE_TEST_DATABASE_URL is not set")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	t.Cleanup(cancel)
	schema := fmt.Sprintf("tmtest_%d", time.Now().UnixNano())
	adminDB, err := sql.Open("pgx", normalizeURL(baseURL))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := adminDB.ExecContext(ctx, `CREATE SCHEMA `+schema); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_, _ = adminDB.ExecContext(context.Background(), `DROP SCHEMA IF EXISTS `+schema+` CASCADE`)
		_ = adminDB.Close()
	})
	schemaURL := withSearchPath(baseURL, schema)
	db, err := sql.Open("pgx", schemaURL)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	if err := goose.SetDialect("postgres"); err != nil {
		t.Fatal(err)
	}
	goose.SetLogger(goose.NopLogger())
	if err := goose.UpContext(ctx, db, migrationsDir(t)); err != nil {
		t.Fatal(err)
	}
	store, err := postgres.Open(ctx, schemaURL, logging.New("ERROR"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(store.Close)
	return store, schemaURL
}

func migrationsDir(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("cannot resolve test helper path")
	}
	return filepath.Clean(filepath.Join(filepath.Dir(file), "..", "..", "migrations"))
}

func normalizeURL(raw string) string {
	return strings.Replace(strings.Replace(raw, "postgresql+asyncpg://", "postgres://", 1), "postgres+asyncpg://", "postgres://", 1)
}

func withSearchPath(raw string, schema string) string {
	raw = normalizeURL(raw)
	parsed, err := url.Parse(raw)
	if err != nil {
		return raw
	}
	values := parsed.Query()
	values.Set("search_path", schema)
	if values.Get("sslmode") == "" {
		values.Set("sslmode", "disable")
	}
	parsed.RawQuery = values.Encode()
	return parsed.String()
}
