package postgres_test

import (
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/tozogabee/restaurant-system/pkg/postgres"
)

// A repository takes postgres.Pool so the same methods work against the pool
// and against an open transaction — which is what lets InTx wrap existing
// repository calls without a second code path. Both implementations are
// asserted here so the property cannot be broken silently by a change to the
// interface.
//
// docs/database/schema-design-template.md Part D6 documents this; the
// assertion is what keeps that documentation true.
func TestPoolAcceptsPoolAndTx(t *testing.T) {
	var _ postgres.Pool = (*pgxpool.Pool)(nil)
	var _ postgres.Pool = (pgx.Tx)(nil)
}
