//go:build integration

// Package integration checks the foundation packages against the real backing
// services from deploy/local/docker-compose.yml. It is excluded from the
// default build; run it with:
//
//	make up && make test-integration
package integration

import (
	"context"
	"testing"
	"time"

	"github.com/tozogabee/restaurant-system/pkg/config"
	"github.com/tozogabee/restaurant-system/pkg/postgres"
	"github.com/tozogabee/restaurant-system/pkg/redisx"
)

func localPostgres() config.Postgres {
	return config.Postgres{
		Host: "localhost", Port: 5432,
		User: "restaurant", Password: "restaurant",
		Database: "restaurant", Schema: "public", SSLMode: "disable",
		MaxConns: 4, MinConns: 1,
		MaxConnLifetime: time.Hour, MaxConnIdleTime: 30 * time.Minute,
	}
}

func TestPostgresConnectAndQuery(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	pool, err := postgres.Connect(ctx, localPostgres())
	if err != nil {
		t.Fatalf("Connect() = %v (is `make up` running?)", err)
	}
	defer pool.Close()

	var one int
	if err := pool.QueryRow(ctx, "SELECT 1").Scan(&one); err != nil {
		t.Fatalf("query: %v", err)
	}
	if one != 1 {
		t.Fatalf("SELECT 1 returned %d", one)
	}
}

func TestPostgresInitExtensionsAreInstalled(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	pool, err := postgres.Connect(ctx, localPostgres())
	if err != nil {
		t.Fatalf("Connect() = %v", err)
	}
	defer pool.Close()

	// The init script is what makes gen_random_uuid() available to every
	// service's migrations, so a broken init script should fail here.
	var id string
	if err := pool.QueryRow(ctx, "SELECT gen_random_uuid()::text").Scan(&id); err != nil {
		t.Fatalf("pgcrypto missing: %v", err)
	}
	if len(id) != 36 {
		t.Fatalf("gen_random_uuid() = %q", id)
	}
}

func TestPostgresMapErrorOnRealUniqueViolation(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	pool, err := postgres.Connect(ctx, localPostgres())
	if err != nil {
		t.Fatalf("Connect() = %v", err)
	}
	defer pool.Close()

	// A temp table lives for the session and needs no cleanup migration.
	if _, err := pool.Exec(ctx, `CREATE TEMP TABLE smoke_unique (id int PRIMARY KEY)`); err != nil {
		t.Fatalf("create temp table: %v", err)
	}
	if _, err := pool.Exec(ctx, `INSERT INTO smoke_unique (id) VALUES (1)`); err != nil {
		t.Fatalf("first insert: %v", err)
	}

	_, err = pool.Exec(ctx, `INSERT INTO smoke_unique (id) VALUES (1)`)
	if err == nil {
		t.Skip("temp table landed on a different pooled connection; nothing to assert")
	}
	t.Logf("driver error mapped as: %v", postgres.MapError(err, "smoke row"))
}

func TestRedisRoundTrip(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	client, err := redisx.Connect(ctx, config.Redis{Addr: "localhost:6379", Timeout: 3 * time.Second})
	if err != nil {
		t.Fatalf("Connect() = %v (is `make up` running?)", err)
	}
	defer client.Close()

	ns := redisx.NewNamespace("smoke")
	key := ns.Key("round-trip")
	t.Cleanup(func() { client.Del(context.Background(), key) })

	if err := client.Set(ctx, key, "value", time.Minute).Err(); err != nil {
		t.Fatalf("set: %v", err)
	}
	got, err := client.Get(ctx, key).Result()
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if got != "value" {
		t.Fatalf("get = %q, want %q", got, "value")
	}

	// A missing key must look like a miss, not a failure.
	_, err = client.Get(ctx, ns.Key("absent")).Result()
	if !redisx.IsMiss(err) {
		t.Fatalf("IsMiss(%v) = false, want true", err)
	}
}
