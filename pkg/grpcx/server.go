// Package grpcx holds the gRPC building blocks shared by services: a server
// with the standard interceptor chain, a dial helper for service-to-service
// calls, and translation between domain errors and gRPC status codes.
package grpcx

import (
	"context"
	"fmt"
	"log/slog"
	"net"

	"google.golang.org/grpc"
	"google.golang.org/grpc/health"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/reflection"
)

// ServerOptions configures NewServer.
type ServerOptions struct {
	Port int
	Log  *slog.Logger
	// Reflection registers the gRPC reflection service so grpcurl and similar
	// tools can introspect the API. Services enable it outside production.
	Reflection bool
	// Extra interceptors run after the standard chain.
	Extra []grpc.UnaryServerInterceptor
}

// Server wraps a grpc.Server with start and graceful-stop semantics matching
// httpx.Server.
type Server struct {
	grpc   *grpc.Server
	health *health.Server
	log    *slog.Logger
	port   int
}

// NewServer builds a gRPC server with request-ID propagation, request logging
// and panic recovery already installed, plus the standard health service.
// Callers register their own service implementations on Registrar.
func NewServer(opts ServerOptions) *Server {
	chain := append([]grpc.UnaryServerInterceptor{
		UnaryRequestID(),
		UnaryLogger(opts.Log),
		UnaryRecovery(),
	}, opts.Extra...)

	srv := grpc.NewServer(grpc.ChainUnaryInterceptor(chain...))

	healthSrv := health.NewServer()
	healthpb.RegisterHealthServer(srv, healthSrv)

	if opts.Reflection {
		reflection.Register(srv)
	}

	return &Server{grpc: srv, health: healthSrv, log: opts.Log, port: opts.Port}
}

// Registrar exposes the underlying server so generated RegisterXServer
// functions can attach a service implementation.
func (s *Server) Registrar() grpc.ServiceRegistrar { return s.grpc }

// Health exposes the health server so a service can flip its serving status
// as dependencies come and go.
func (s *Server) Health() *health.Server { return s.health }

// Serve blocks until the server stops.
func (s *Server) Serve() error {
	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", s.port))
	if err != nil {
		return fmt.Errorf("grpc listen on %d: %w", s.port, err)
	}

	s.log.Info("grpc server listening", slog.Int("port", s.port))
	if err := s.grpc.Serve(lis); err != nil {
		return fmt.Errorf("grpc server: %w", err)
	}
	return nil
}

// Stop finishes in-flight RPCs and then stops. It falls back to a hard stop
// when ctx is done first, so a stuck stream cannot block shutdown forever.
func (s *Server) Stop(ctx context.Context) error {
	s.log.Info("grpc server shutting down")
	s.health.Shutdown()

	stopped := make(chan struct{})
	go func() {
		s.grpc.GracefulStop()
		close(stopped)
	}()

	select {
	case <-stopped:
		return nil
	case <-ctx.Done():
		s.grpc.Stop()
		return fmt.Errorf("grpc graceful stop: %w", ctx.Err())
	}
}
