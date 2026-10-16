package grpcx

import (
	"context"
	"fmt"
	"log/slog"
	"runtime/debug"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"

	"github.com/tozogabee/restaurant-system/pkg/errs"
	"github.com/tozogabee/restaurant-system/pkg/logger"
	"github.com/tozogabee/restaurant-system/pkg/reqid"
)

// UnaryRequestID adopts the request ID from inbound metadata, or generates one
// when the caller sent none, and puts it on the handler's context.
func UnaryRequestID() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, _ *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		if md, ok := metadata.FromIncomingContext(ctx); ok {
			if values := md.Get(reqid.Header); len(values) > 0 {
				ctx = reqid.Into(ctx, values[0])
			}
		}
		ctx, _ = reqid.Ensure(ctx)
		return handler(ctx, req)
	}
}

// UnaryLogger puts a request-scoped logger on the context and logs one line
// per finished RPC. It must run after UnaryRequestID.
func UnaryLogger(log *slog.Logger) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		rpcLog := log.With(
			slog.String("request_id", reqid.From(ctx)),
			slog.String("rpc", info.FullMethod),
		)
		ctx = logger.Into(ctx, rpcLog)

		start := time.Now()
		resp, err := handler(ctx, req)
		duration := time.Since(start)

		if err != nil {
			kind := errs.KindOf(err)
			level := slog.LevelWarn
			if kind == errs.KindInternal {
				level = slog.LevelError
			}
			rpcLog.Log(ctx, level, "rpc failed",
				slog.Any("error", err),
				slog.String("kind", string(kind)),
				slog.Duration("duration", duration),
			)
			return resp, Status(err)
		}

		rpcLog.InfoContext(ctx, "rpc handled", slog.Duration("duration", duration))
		return resp, nil
	}
}

// UnaryRecovery turns a panic in a handler into an internal gRPC error.
func UnaryRecovery() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, _ *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (resp any, err error) {
		defer func() {
			if recovered := recover(); recovered != nil {
				logger.From(ctx).ErrorContext(ctx, "panic recovered",
					slog.Any("panic", recovered),
					slog.String("stack", string(debug.Stack())),
				)
				resp = nil
				err = Status(errs.Internal("panic").Wrap(fmt.Errorf("%v", recovered)))
			}
		}()
		return handler(ctx, req)
	}
}

// UnaryClientRequestID copies the request ID from the calling context into
// outbound metadata, so a chain of service calls shares one correlation ID.
func UnaryClientRequestID() grpc.UnaryClientInterceptor {
	return func(ctx context.Context, method string, req, reply any, cc *grpc.ClientConn, invoker grpc.UnaryInvoker, opts ...grpc.CallOption) error {
		if id := reqid.From(ctx); id != "" {
			ctx = metadata.AppendToOutgoingContext(ctx, reqid.Header, id)
		}
		return invoker(ctx, method, req, reply, cc, opts...)
	}
}
