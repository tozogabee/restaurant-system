// Package httpx holds the Gin building blocks shared by every service that
// exposes HTTP: middleware, error rendering and a server with graceful
// shutdown. Services register their own routes on the engine it builds.
package httpx

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/tozogabee/restaurant-system/pkg/errs"
	"github.com/tozogabee/restaurant-system/pkg/logger"
	"github.com/tozogabee/restaurant-system/pkg/reqid"
)

// ErrorBody is the single error shape every service returns, so one frontend
// error handler covers all of them.
type ErrorBody struct {
	Error ErrorDetail `json:"error"`
}

// ErrorDetail describes what went wrong.
type ErrorDetail struct {
	// Kind is the coarse classification, e.g. "not_found".
	Kind string `json:"kind"`
	// Code is a stable identifier to branch on, when the service set one.
	Code string `json:"code,omitempty"`
	// Message is a human-readable description, safe to show to a user.
	Message string `json:"message"`
	// Fields maps field names to validation problems.
	Fields map[string]string `json:"fields,omitempty"`
	// RequestID correlates this response with the service logs.
	RequestID string `json:"request_id,omitempty"`
}

// statusFor maps a domain error kind onto an HTTP status code.
func statusFor(kind errs.Kind) int {
	switch kind {
	case errs.KindInvalidInput:
		return http.StatusBadRequest
	case errs.KindUnauthorized:
		return http.StatusUnauthorized
	case errs.KindForbidden:
		return http.StatusForbidden
	case errs.KindNotFound:
		return http.StatusNotFound
	case errs.KindConflict:
		return http.StatusConflict
	case errs.KindUnavailable:
		return http.StatusServiceUnavailable
	default:
		return http.StatusInternalServerError
	}
}

// Fail aborts the request with the status implied by err and logs it. Handlers
// end every failure path with this call so the response shape, the status code
// and the log entry stay in one place.
//
// Internal errors are logged at error level and reported to the client as a
// generic message; everything else is logged at warn level with its own
// message passed through.
func Fail(c *gin.Context, err error) {
	ctx := c.Request.Context()
	kind := errs.KindOf(err)
	status := statusFor(kind)

	body := ErrorBody{Error: ErrorDetail{
		Kind:      string(kind),
		RequestID: reqid.From(ctx),
	}}

	if domain := errs.As(err); domain != nil {
		body.Error.Code = domain.Code
		body.Error.Message = domain.Message
		body.Error.Fields = domain.Fields
	}

	log := logger.From(ctx)
	if kind == errs.KindInternal {
		// Never leak an internal message or a wrapped cause to the client.
		body.Error.Message = "internal server error"
		body.Error.Fields = nil
		log.ErrorContext(ctx, "request failed", "error", err, "status", status)
	} else {
		if body.Error.Message == "" {
			body.Error.Message = string(kind)
		}
		log.WarnContext(ctx, "request rejected", "error", err, "status", status)
	}

	c.AbortWithStatusJSON(status, body)
}

// OK writes a 200 with the given payload.
func OK(c *gin.Context, payload any) { c.JSON(http.StatusOK, payload) }

// Created writes a 201 with the given payload.
func Created(c *gin.Context, payload any) { c.JSON(http.StatusCreated, payload) }

// NoContent writes a 204.
func NoContent(c *gin.Context) { c.Status(http.StatusNoContent) }

// Bind decodes and validates the JSON request body into dst, reporting a
// domain InvalidInput error on failure so the caller can hand it to Fail.
func Bind(c *gin.Context, dst any) error {
	if err := c.ShouldBindJSON(dst); err != nil {
		return errs.InvalidInput("malformed request body").Wrap(err)
	}
	return nil
}
