package errs_test

import (
	"errors"
	"fmt"
	"testing"

	"github.com/tozogabee/restaurant-system/pkg/errs"
)

func TestKindOfUntypedErrorIsInternal(t *testing.T) {
	if got := errs.KindOf(errors.New("boom")); got != errs.KindInternal {
		t.Fatalf("KindOf(untyped) = %q, want %q", got, errs.KindInternal)
	}
}

func TestKindOfSurvivesWrapping(t *testing.T) {
	err := fmt.Errorf("service layer: %w", errs.NotFound("menu item not found"))

	if got := errs.KindOf(err); got != errs.KindNotFound {
		t.Fatalf("KindOf(wrapped) = %q, want %q", got, errs.KindNotFound)
	}
	if !errs.Is(err, errs.KindNotFound) {
		t.Fatal("Is(err, KindNotFound) = false, want true")
	}
}

func TestCauseIsReachableButNotInMessage(t *testing.T) {
	cause := errors.New("connection refused")
	err := errs.Unavailable("menu service unreachable").Wrap(cause)

	if !errors.Is(err, cause) {
		t.Fatal("errors.Is(err, cause) = false; the cause must stay reachable for logs")
	}

	domain := errs.As(err)
	if domain == nil {
		t.Fatal("As(err) = nil, want the domain error")
	}
	if domain.Message != "menu service unreachable" {
		t.Fatalf("Message = %q; the client-safe message must not absorb the cause", domain.Message)
	}
}

func TestWithFieldAccumulates(t *testing.T) {
	err := errs.InvalidInput("invalid reservation").
		WithCode("reservation_invalid").
		WithField("party_size", "must be at least 1").
		WithField("starts_at", "must be in the future")

	if err.Code != "reservation_invalid" {
		t.Fatalf("Code = %q, want %q", err.Code, "reservation_invalid")
	}
	if len(err.Fields) != 2 {
		t.Fatalf("len(Fields) = %d, want 2", len(err.Fields))
	}
	if err.Fields["party_size"] != "must be at least 1" {
		t.Fatalf("Fields[party_size] = %q", err.Fields["party_size"])
	}
}
