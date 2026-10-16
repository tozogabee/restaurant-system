package config_test

import (
	"testing"
	"time"

	"github.com/tozogabee/restaurant-system/pkg/config"
)

type serviceConfig struct {
	config.Base
	Postgres config.Postgres
	Redis    config.Redis
}

func TestLoadAppliesDefaultsAndParsesTypes(t *testing.T) {
	t.Setenv("SERVICE_NAME", "menu")
	t.Setenv("HTTP_PORT", "8081")
	t.Setenv("POSTGRES_USER", "restaurant")
	t.Setenv("POSTGRES_PASSWORD", "secret")
	t.Setenv("POSTGRES_DB", "restaurant")
	t.Setenv("POSTGRES_SCHEMA", "menu")

	cfg, err := config.Load[serviceConfig]()
	if err != nil {
		t.Fatalf("Load() = %v", err)
	}

	if cfg.ServiceName != "menu" || cfg.HTTPPort != 8081 {
		t.Fatalf("base = %+v", cfg.Base)
	}
	if cfg.Environment != config.EnvLocal {
		t.Errorf("Environment = %q, want the local default", cfg.Environment)
	}
	if cfg.ShutdownTimeout != 15*time.Second {
		t.Errorf("ShutdownTimeout = %s, want 15s", cfg.ShutdownTimeout)
	}
	if cfg.Postgres.Port != 5432 {
		t.Errorf("Postgres.Port = %d, want the 5432 default", cfg.Postgres.Port)
	}
	if cfg.Redis.Addr != "localhost:6379" {
		t.Errorf("Redis.Addr = %q, want the local default", cfg.Redis.Addr)
	}
	if cfg.IsProd() {
		t.Error("IsProd() = true for the local environment")
	}
}

func TestLoadRequiresServiceName(t *testing.T) {
	t.Setenv("POSTGRES_USER", "restaurant")
	t.Setenv("POSTGRES_PASSWORD", "secret")
	t.Setenv("POSTGRES_DB", "restaurant")
	t.Setenv("POSTGRES_SCHEMA", "menu")

	// SERVICE_NAME is deliberately unset: a service must not start without
	// knowing its own name, because it tags every log line and every metric.
	if _, err := config.Load[serviceConfig](); err == nil {
		t.Fatal("Load() = nil error, want a failure for the missing SERVICE_NAME")
	}
}

func TestPostgresDSNPinsSearchPathToTheServiceSchema(t *testing.T) {
	dsn := config.Postgres{
		Host: "db", Port: 5432,
		User: "restaurant", Password: "secret",
		Database: "restaurant", Schema: "menu", SSLMode: "disable",
	}.DSN()

	want := "postgres://restaurant:secret@db:5432/restaurant?sslmode=disable&search_path=menu"
	if dsn != want {
		t.Fatalf("DSN() =\n  %s\nwant\n  %s", dsn, want)
	}
}
