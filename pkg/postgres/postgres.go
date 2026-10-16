// Package postgres opens the pgx connection pool a service uses to reach its
// own schema. Each service owns one schema and never reads another's tables.
package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/tozogabee/restaurant-system/pkg/config"
	"github.com/tozogabee/restaurant-system/pkg/errs"
)

// Pool is the subset of pgxpool.Pool that repositories depend on. Taking this
// interface instead of the concrete type lets a repository be tested against a
// fake, and lets a method accept either the pool or an open transaction.
type Pool interface {
	Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}

// Connect opens a pool and verifies it with a ping, so a service fails at
// startup rather than on its first request.
func Connect(ctx context.Context, cfg config.Postgres) (*pgxpool.Pool, error) {
	poolCfg, err := pgxpool.ParseConfig(cfg.DSN())
	if err != nil {
		return nil, fmt.Errorf("parse postgres dsn: %w", err)
	}

	poolCfg.MaxConns = cfg.MaxConns
	poolCfg.MinConns = cfg.MinConns
	poolCfg.MaxConnLifetime = cfg.MaxConnLifetime
	poolCfg.MaxConnIdleTime = cfg.MaxConnIdleTime

	pool, err := pgxpool.NewWithConfig(ctx, poolCfg)
	if err != nil {
		return nil, fmt.Errorf("open postgres pool: %w", err)
	}

	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("ping postgres: %w", err)
	}

	return pool, nil
}

// Postgres error codes that carry a clear domain meaning.
// See https://www.postgresql.org/docs/current/errcodes-appendix.html
const (
	codeUniqueViolation     = "23505"
	codeForeignKeyViolation = "23503"
	codeCheckViolation      = "23514"
	codeNotNullViolation    = "23502"
)

// MapError translates a pgx error into a domain error. Repositories pass every
// driver error through it so the transport layer sees a Kind, not a pg code.
//
// A missing row becomes KindNotFound with a generic message; pass the resource
// name so the message tells the client what was missing.
func MapError(err error, resource string) error {
	if err == nil {
		return nil
	}

	if errors.Is(err, pgx.ErrNoRows) {
		return errs.NotFound("%s not found", resource).Wrap(err)
	}

	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		switch pgErr.Code {
		case codeUniqueViolation:
			return errs.Conflict("%s already exists", resource).Wrap(err)
		case codeForeignKeyViolation:
			return errs.InvalidInput("%s references something that does not exist", resource).Wrap(err)
		case codeCheckViolation, codeNotNullViolation:
			return errs.InvalidInput("%s is not valid", resource).Wrap(err)
		}
	}

	return errs.Internal("%s query failed", resource).Wrap(err)
}
