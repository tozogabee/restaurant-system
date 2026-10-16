package httpx

import (
	"fmt"
	"log/slog"
	"runtime/debug"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/tozogabee/restaurant-system/pkg/errs"
	"github.com/tozogabee/restaurant-system/pkg/logger"
	"github.com/tozogabee/restaurant-system/pkg/reqid"
)

// RequestID adopts an inbound request ID or generates one, stores it on the
// request context for downstream calls, and echoes it back in the response so
// a client can quote it in a bug report.
func RequestID() gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.GetHeader(reqid.Header)
		if id == "" {
			id = reqid.New()
		}
		c.Request = c.Request.WithContext(reqid.Into(c.Request.Context(), id))
		c.Header(reqid.Header, id)
		c.Next()
	}
}

// Logger puts a request-scoped logger on the context and logs one line per
// finished request. It must run after RequestID to pick up the correlation ID.
func Logger(log *slog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := c.Request.Context()
		reqLog := log.With(
			slog.String("request_id", reqid.From(ctx)),
			slog.String("method", c.Request.Method),
			slog.String("path", c.FullPath()),
		)
		c.Request = c.Request.WithContext(logger.Into(ctx, reqLog))

		start := time.Now()
		c.Next()

		// Failures are already logged with their cause by Fail, so this line
		// only records the outcome and timing.
		reqLog.InfoContext(c.Request.Context(), "request handled",
			slog.Int("status", c.Writer.Status()),
			slog.Duration("duration", time.Since(start)),
			slog.String("client_ip", c.ClientIP()),
		)
	}
}

// Recovery turns a panic into the standard 500 error body. It must be
// registered before any handler that could panic.
func Recovery() gin.HandlerFunc {
	return gin.CustomRecoveryWithWriter(nil, func(c *gin.Context, recovered any) {
		ctx := c.Request.Context()
		logger.From(ctx).ErrorContext(ctx, "panic recovered",
			slog.Any("panic", recovered),
			slog.String("stack", string(debug.Stack())),
		)
		Fail(c, errs.Internal("panic").Wrap(fmt.Errorf("%v", recovered)))
	})
}
