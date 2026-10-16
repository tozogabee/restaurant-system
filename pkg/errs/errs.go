// Package errs defines the error vocabulary shared by every service.
//
// Domain and repository code returns an *Error with a Kind; the HTTP and gRPC
// layers translate that Kind into a status code. Nothing below the transport
// layer needs to know about HTTP or gRPC.
package errs

import (
	"errors"
	"fmt"
)

// Kind classifies an error well enough for a transport to pick a status code.
type Kind string

const (
	KindInvalidInput Kind = "invalid_input"
	KindUnauthorized Kind = "unauthorized"
	KindForbidden    Kind = "forbidden"
	KindNotFound     Kind = "not_found"
	KindConflict     Kind = "conflict"
	KindUnavailable  Kind = "unavailable"
	KindInternal     Kind = "internal"
)

// Error is a domain error. Message is safe to return to clients; the wrapped
// cause is for logs only.
type Error struct {
	Kind Kind
	// Code is an optional stable identifier for clients to branch on,
	// e.g. "menu_item_out_of_stock".
	Code string
	// Message is a client-safe description.
	Message string
	// Fields carries per-field validation details, keyed by field name.
	Fields map[string]string

	cause error
}

func (e *Error) Error() string {
	if e.cause != nil {
		return fmt.Sprintf("%s: %s: %v", e.Kind, e.Message, e.cause)
	}
	return fmt.Sprintf("%s: %s", e.Kind, e.Message)
}

// Unwrap exposes the underlying cause to errors.Is and errors.As.
func (e *Error) Unwrap() error { return e.cause }

// WithCode attaches a stable client-facing code and returns e.
func (e *Error) WithCode(code string) *Error {
	e.Code = code
	return e
}

// WithField records a validation problem for a single field and returns e.
func (e *Error) WithField(name, problem string) *Error {
	if e.Fields == nil {
		e.Fields = make(map[string]string)
	}
	e.Fields[name] = problem
	return e
}

// Wrap attaches cause to e and returns e. The cause is logged, never returned
// to clients.
func (e *Error) Wrap(cause error) *Error {
	e.cause = cause
	return e
}

func newf(kind Kind, format string, args ...any) *Error {
	return &Error{Kind: kind, Message: fmt.Sprintf(format, args...)}
}

// InvalidInput reports a malformed or semantically invalid request.
func InvalidInput(format string, args ...any) *Error {
	return newf(KindInvalidInput, format, args...)
}

// Unauthorized reports missing or invalid credentials.
func Unauthorized(format string, args ...any) *Error {
	return newf(KindUnauthorized, format, args...)
}

// Forbidden reports valid credentials without sufficient permission.
func Forbidden(format string, args ...any) *Error {
	return newf(KindForbidden, format, args...)
}

// NotFound reports a missing resource.
func NotFound(format string, args ...any) *Error {
	return newf(KindNotFound, format, args...)
}

// Conflict reports a clash with current state, such as a uniqueness violation
// or a failed optimistic-locking check.
func Conflict(format string, args ...any) *Error {
	return newf(KindConflict, format, args...)
}

// Unavailable reports a dependency that is down or timed out.
func Unavailable(format string, args ...any) *Error {
	return newf(KindUnavailable, format, args...)
}

// Internal reports an unexpected failure. The message is deliberately generic;
// put the detail in the wrapped cause.
func Internal(format string, args ...any) *Error {
	return newf(KindInternal, format, args...)
}

// KindOf returns the Kind of the first *Error in err's chain, or KindInternal
// when there is none. An untyped error is always treated as internal.
func KindOf(err error) Kind {
	var e *Error
	if errors.As(err, &e) {
		return e.Kind
	}
	return KindInternal
}

// As returns the first *Error in err's chain, or nil.
func As(err error) *Error {
	var e *Error
	if errors.As(err, &e) {
		return e
	}
	return nil
}

// Is reports whether err's chain contains an *Error of the given kind.
func Is(err error, kind Kind) bool { return KindOf(err) == kind }
