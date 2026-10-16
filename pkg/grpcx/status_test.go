package grpcx_test

import (
	"errors"
	"testing"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/tozogabee/restaurant-system/pkg/errs"
	"github.com/tozogabee/restaurant-system/pkg/grpcx"
)

func TestStatusMapsEveryKind(t *testing.T) {
	cases := []struct {
		err  error
		want codes.Code
	}{
		{errs.InvalidInput("bad"), codes.InvalidArgument},
		{errs.Unauthorized("no token"), codes.Unauthenticated},
		{errs.Forbidden("not yours"), codes.PermissionDenied},
		{errs.NotFound("gone"), codes.NotFound},
		{errs.Conflict("taken"), codes.AlreadyExists},
		{errs.Unavailable("down"), codes.Unavailable},
		{errs.Internal("oops"), codes.Internal},
		{errors.New("untyped"), codes.Internal},
	}

	for _, tc := range cases {
		got := status.Code(grpcx.Status(tc.err))
		if got != tc.want {
			t.Errorf("Status(%v) code = %v, want %v", tc.err, got, tc.want)
		}
	}
}

func TestStatusHidesInternalDetail(t *testing.T) {
	err := grpcx.Status(errs.Internal("query failed").Wrap(errors.New("pq: relation \"orders\" does not exist")))

	if msg := status.Convert(err).Message(); msg != "internal error" {
		t.Fatalf("message = %q; an internal cause must not cross the service boundary", msg)
	}
}

func TestStatusNilStaysNil(t *testing.T) {
	if err := grpcx.Status(nil); err != nil {
		t.Fatalf("Status(nil) = %v, want nil", err)
	}
}

func TestFromStatusRestoresKind(t *testing.T) {
	// A server rejecting a call with NotFound must reach the caller as a
	// domain NotFound, not as an opaque internal error.
	wire := status.Error(codes.NotFound, "menu item not found")

	err := grpcx.FromStatus(wire)
	if !errs.Is(err, errs.KindNotFound) {
		t.Fatalf("FromStatus kind = %q, want %q", errs.KindOf(err), errs.KindNotFound)
	}
	if domain := errs.As(err); domain == nil || domain.Message != "menu item not found" {
		t.Fatalf("message not carried through: %+v", domain)
	}
}

func TestFromStatusRoundTripsThroughStatus(t *testing.T) {
	for _, kind := range []errs.Kind{
		errs.KindInvalidInput,
		errs.KindUnauthorized,
		errs.KindForbidden,
		errs.KindNotFound,
		errs.KindConflict,
		errs.KindUnavailable,
		errs.KindInternal,
	} {
		original := &errs.Error{Kind: kind, Message: "x"}
		got := errs.KindOf(grpcx.FromStatus(grpcx.Status(original)))
		if got != kind {
			t.Errorf("round trip of %q gave %q", kind, got)
		}
	}
}
