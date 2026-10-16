// Package reqid propagates a request correlation ID across HTTP and gRPC hops
// so one client call can be traced through every service that handled it.
package reqid

import (
	"context"

	"github.com/google/uuid"
)

// Header is the HTTP header and gRPC metadata key carrying the ID. gRPC
// metadata keys are lowercase, so the same string works for both.
const Header = "x-request-id"

type ctxKey struct{}

// Into returns a copy of ctx carrying id. An empty id is ignored.
func Into(ctx context.Context, id string) context.Context {
	if id == "" {
		return ctx
	}
	return context.WithValue(ctx, ctxKey{}, id)
}

// From returns the ID stored in ctx, or "" when there is none.
func From(ctx context.Context) string {
	id, _ := ctx.Value(ctxKey{}).(string)
	return id
}

// New generates a fresh request ID.
func New() string { return uuid.NewString() }

// Ensure returns the ID already in ctx, or stores and returns a new one.
func Ensure(ctx context.Context) (context.Context, string) {
	if id := From(ctx); id != "" {
		return ctx, id
	}
	id := New()
	return Into(ctx, id), id
}
