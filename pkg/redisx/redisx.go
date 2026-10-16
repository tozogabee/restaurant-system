// Package redisx opens the Redis client used for caching and session storage.
//
// Keys are namespaced per service via Namespaced so two services sharing one
// Redis instance cannot collide.
package redisx

import (
	"context"
	"errors"
	"fmt"

	"github.com/redis/go-redis/v9"

	"github.com/tozogabee/restaurant-system/pkg/config"
	"github.com/tozogabee/restaurant-system/pkg/errs"
)

// Connect opens a client and verifies it with a ping.
func Connect(ctx context.Context, cfg config.Redis) (*redis.Client, error) {
	client := redis.NewClient(&redis.Options{
		Addr:         cfg.Addr,
		Password:     cfg.Password,
		DB:           cfg.DB,
		DialTimeout:  cfg.Timeout,
		ReadTimeout:  cfg.Timeout,
		WriteTimeout: cfg.Timeout,
	})

	if err := client.Ping(ctx).Err(); err != nil {
		_ = client.Close()
		return nil, fmt.Errorf("ping redis: %w", err)
	}

	return client, nil
}

// Namespaced prefixes keys with a service name, e.g. Namespaced("menu") then
// ns.Key("item", id) yields "menu:item:<id>".
type Namespaced struct {
	prefix string
}

// NewNamespace returns a key builder scoped to service.
func NewNamespace(service string) Namespaced { return Namespaced{prefix: service} }

// Key joins the namespace and the given parts with colons.
func (n Namespaced) Key(parts ...string) string {
	key := n.prefix
	for _, part := range parts {
		key += ":" + part
	}
	return key
}

// IsMiss reports whether err is a plain cache miss rather than a failure.
// Callers treat a miss as "not cached" and fall through to the source of
// truth; anything else is a real error.
func IsMiss(err error) bool { return errors.Is(err, redis.Nil) }

// MapError translates a Redis error into a domain error, reporting a cache
// miss as KindNotFound. A cache is a soft dependency: callers that can fall
// back to the database should check IsMiss and ignore failures instead.
func MapError(err error, op string) error {
	if err == nil {
		return nil
	}
	if IsMiss(err) {
		return errs.NotFound("cache miss").Wrap(err)
	}
	return errs.Unavailable("redis %s failed", op).Wrap(err)
}
