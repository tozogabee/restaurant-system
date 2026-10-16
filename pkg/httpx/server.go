package httpx

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

// ServerOptions configures the engine and server built by NewEngine/NewServer.
type ServerOptions struct {
	Port int
	Log  *slog.Logger
	// Debug keeps Gin in debug mode and its verbose startup output. Services
	// pass true only for local development.
	Debug bool

	ReadTimeout  time.Duration
	WriteTimeout time.Duration
	IdleTimeout  time.Duration
}

func (o ServerOptions) withDefaults() ServerOptions {
	if o.ReadTimeout == 0 {
		o.ReadTimeout = 10 * time.Second
	}
	if o.WriteTimeout == 0 {
		o.WriteTimeout = 15 * time.Second
	}
	if o.IdleTimeout == 0 {
		o.IdleTimeout = 60 * time.Second
	}
	return o
}

// NewEngine returns a Gin engine with the standard middleware chain already
// registered, in the order the middleware requires: RequestID, then Logger,
// then Recovery. Callers add their own routes and groups.
func NewEngine(opts ServerOptions) *gin.Engine {
	if opts.Debug {
		gin.SetMode(gin.DebugMode)
	} else {
		gin.SetMode(gin.ReleaseMode)
	}

	e := gin.New()
	e.Use(RequestID(), Logger(opts.Log), Recovery())
	return e
}

// Server wraps an http.Server with start and graceful-stop semantics that
// match grpcx.Server, so a service can run both from the same supervisor.
type Server struct {
	http *http.Server
	log  *slog.Logger
	port int
}

// NewServer wraps handler in a configured http.Server.
func NewServer(handler http.Handler, opts ServerOptions) *Server {
	opts = opts.withDefaults()
	return &Server{
		http: &http.Server{
			Addr:         fmt.Sprintf(":%d", opts.Port),
			Handler:      handler,
			ReadTimeout:  opts.ReadTimeout,
			WriteTimeout: opts.WriteTimeout,
			IdleTimeout:  opts.IdleTimeout,
		},
		log:  opts.Log,
		port: opts.Port,
	}
}

// Serve blocks until the server stops. A shutdown triggered by Stop is not an
// error and returns nil.
func (s *Server) Serve() error {
	s.log.Info("http server listening", slog.Int("port", s.port))
	if err := s.http.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		return fmt.Errorf("http server: %w", err)
	}
	return nil
}

// Stop drains in-flight requests, giving up when ctx is done.
func (s *Server) Stop(ctx context.Context) error {
	s.log.Info("http server shutting down")
	if err := s.http.Shutdown(ctx); err != nil {
		return fmt.Errorf("http shutdown: %w", err)
	}
	return nil
}
