// Package app runs the long-lived parts of a service — an HTTP server, a gRPC
// server, a background worker — under one supervisor, and shuts them all down
// cleanly on SIGINT or SIGTERM.
//
// A service's main is then roughly:
//
//	runner := app.New(cfg.ShutdownTimeout, log)
//	runner.Add("http", httpSrv.Serve, httpSrv.Stop)
//	runner.Add("grpc", grpcSrv.Serve, grpcSrv.Stop)
//	runner.AddCloser("postgres", closePool)
//	if err := runner.Run(ctx); err != nil { ... }
package app

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os/signal"
	"syscall"
	"time"
)

// ServeFunc blocks until its component stops. Returning nil means a clean
// stop; returning an error brings the whole service down.
type ServeFunc func() error

// StopFunc stops a component, giving up when ctx is done.
type StopFunc func(ctx context.Context) error

type component struct {
	name  string
	serve ServeFunc
	stop  StopFunc
}

// Runner supervises a service's components.
type Runner struct {
	components      []component
	shutdownTimeout time.Duration
	log             *slog.Logger
}

// New returns a Runner. A zero shutdownTimeout defaults to 15s.
func New(shutdownTimeout time.Duration, log *slog.Logger) *Runner {
	if shutdownTimeout == 0 {
		shutdownTimeout = 15 * time.Second
	}
	return &Runner{shutdownTimeout: shutdownTimeout, log: log}
}

// Add registers a component with both a blocking serve and a stop.
func (r *Runner) Add(name string, serve ServeFunc, stop StopFunc) {
	r.components = append(r.components, component{name: name, serve: serve, stop: stop})
}

// AddCloser registers something that only needs closing on shutdown, such as a
// database pool or a cache client.
func (r *Runner) AddCloser(name string, stop StopFunc) {
	r.components = append(r.components, component{name: name, stop: stop})
}

// Run starts every component and blocks until a signal arrives, ctx is
// cancelled, or a component fails. It then stops all of them in reverse
// registration order, so dependents shut down before their dependencies.
//
// The returned error is the first failure seen, whether from serving or from
// shutdown; a clean signal-triggered stop returns nil.
func (r *Runner) Run(ctx context.Context) error {
	ctx, stopSignals := signal.NotifyContext(ctx, syscall.SIGINT, syscall.SIGTERM)
	defer stopSignals()

	// Buffered so a component that fails after shutdown has begun can still
	// report it without blocking on a receiver that has moved on.
	failures := make(chan error, len(r.components))

	for _, c := range r.components {
		if c.serve == nil {
			continue
		}
		go func(c component) {
			if err := c.serve(); err != nil {
				failures <- fmt.Errorf("%s: %w", c.name, err)
			}
		}(c)
	}

	var runErr error
	select {
	case <-ctx.Done():
		r.log.Info("shutdown signal received")
	case runErr = <-failures:
		r.log.Error("component failed, shutting down", slog.Any("error", runErr))
	}

	return errors.Join(runErr, r.shutdown())
}

// shutdown stops components in reverse order under a single deadline shared by
// all of them, so a slow component cannot extend total shutdown time.
func (r *Runner) shutdown() error {
	ctx, cancel := context.WithTimeout(context.Background(), r.shutdownTimeout)
	defer cancel()

	var errs []error
	for i := len(r.components) - 1; i >= 0; i-- {
		c := r.components[i]
		if c.stop == nil {
			continue
		}
		if err := c.stop(ctx); err != nil {
			r.log.Error("component shutdown failed",
				slog.String("component", c.name),
				slog.Any("error", err),
			)
			errs = append(errs, fmt.Errorf("stop %s: %w", c.name, err))
		}
	}

	r.log.Info("shutdown complete")
	return errors.Join(errs...)
}
