package grpcx

import (
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/tozogabee/restaurant-system/pkg/errs"
)

// codeFor maps a domain error kind onto a gRPC status code.
func codeFor(kind errs.Kind) codes.Code {
	switch kind {
	case errs.KindInvalidInput:
		return codes.InvalidArgument
	case errs.KindUnauthorized:
		return codes.Unauthenticated
	case errs.KindForbidden:
		return codes.PermissionDenied
	case errs.KindNotFound:
		return codes.NotFound
	case errs.KindConflict:
		return codes.AlreadyExists
	case errs.KindUnavailable:
		return codes.Unavailable
	default:
		return codes.Internal
	}
}

// kindFor maps a gRPC status code back onto a domain error kind. It is the
// inverse of codeFor and is used when a client call fails, so an upstream
// service's error keeps its meaning as it travels.
func kindFor(code codes.Code) errs.Kind {
	switch code {
	case codes.InvalidArgument, codes.OutOfRange, codes.FailedPrecondition:
		return errs.KindInvalidInput
	case codes.Unauthenticated:
		return errs.KindUnauthorized
	case codes.PermissionDenied:
		return errs.KindForbidden
	case codes.NotFound:
		return errs.KindNotFound
	case codes.AlreadyExists, codes.Aborted:
		return errs.KindConflict
	case codes.Unavailable, codes.DeadlineExceeded, codes.ResourceExhausted:
		return errs.KindUnavailable
	default:
		return errs.KindInternal
	}
}

// Status converts a domain error into a gRPC error for a server to return.
// Internal errors are reported with a generic message so a wrapped cause never
// crosses a service boundary; the caller is expected to have logged it.
func Status(err error) error {
	if err == nil {
		return nil
	}

	kind := errs.KindOf(err)
	code := codeFor(kind)

	if kind == errs.KindInternal {
		return status.Error(code, "internal error")
	}

	msg := string(kind)
	if domain := errs.As(err); domain != nil && domain.Message != "" {
		msg = domain.Message
		if domain.Code != "" {
			msg = domain.Code + ": " + msg
		}
	}
	return status.Error(code, msg)
}

// FromStatus converts an error returned by a gRPC client call back into a
// domain error, so callers can classify it with errs.Is instead of inspecting
// gRPC codes.
func FromStatus(err error) error {
	if err == nil {
		return nil
	}

	st, ok := status.FromError(err)
	if !ok {
		return errs.Internal("grpc call failed").Wrap(err)
	}

	kind := kindFor(st.Code())
	msg := st.Message()
	if kind == errs.KindInternal {
		msg = "upstream service error"
	}

	return (&errs.Error{Kind: kind, Message: msg}).Wrap(err)
}
