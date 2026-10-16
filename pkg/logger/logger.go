// Package logger builds the structured logger used across services and carries
// it through context so handlers, services and repositories all log with the
// same request-scoped fields.
package logger

import (
	"context"
	"log/slog"
	"os"
	"strings"
)

type ctxKey struct{}

// Options configures the logger returned by New.
type Options struct {
	// ServiceName is attached to every record as "service".
	ServiceName string
	// Level is one of debug, info, warn, error. Unknown values fall back to info.
	Level string
	// JSON selects the JSON handler over the human-readable text handler.
	// Services pass true for anything other than local development.
	JSON bool
}

// New returns a logger that writes to stderr.
func New(opts Options) *slog.Logger {
	handlerOpts := &slog.HandlerOptions{Level: parseLevel(opts.Level)}

	var h slog.Handler
	if opts.JSON {
		h = slog.NewJSONHandler(os.Stderr, handlerOpts)
	} else {
		h = slog.NewTextHandler(os.Stderr, handlerOpts)
	}

	log := slog.New(h)
	if opts.ServiceName != "" {
		log = log.With(slog.String("service", opts.ServiceName))
	}
	return log
}

func parseLevel(level string) slog.Level {
	switch strings.ToLower(strings.TrimSpace(level)) {
	case "debug":
		return slog.LevelDebug
	case "warn", "warning":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}

// Into returns a copy of ctx carrying log.
func Into(ctx context.Context, log *slog.Logger) context.Context {
	return context.WithValue(ctx, ctxKey{}, log)
}

// From returns the logger stored in ctx, or the default logger when there is
// none. It never returns nil, so callers can log unconditionally.
func From(ctx context.Context) *slog.Logger {
	if log, ok := ctx.Value(ctxKey{}).(*slog.Logger); ok && log != nil {
		return log
	}
	return slog.Default()
}

// With returns a context whose logger has the given attributes attached.
func With(ctx context.Context, attrs ...slog.Attr) context.Context {
	args := make([]any, len(attrs))
	for i, a := range attrs {
		args[i] = a
	}
	return Into(ctx, From(ctx).With(args...))
}
