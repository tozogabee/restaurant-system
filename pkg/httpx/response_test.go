package httpx_test

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"

	"github.com/tozogabee/restaurant-system/pkg/errs"
	"github.com/tozogabee/restaurant-system/pkg/httpx"
	"github.com/tozogabee/restaurant-system/pkg/logger"
	"github.com/tozogabee/restaurant-system/pkg/reqid"
)

func TestMain(m *testing.M) {
	gin.SetMode(gin.TestMode)
	m.Run()
}

// serve runs handler through the standard middleware chain and returns the
// recorded response.
func serve(t *testing.T, method, path string, handler gin.HandlerFunc) *httptest.ResponseRecorder {
	t.Helper()

	log := logger.New(logger.Options{ServiceName: "test", Level: "error"})
	engine := httpx.NewEngine(httpx.ServerOptions{Log: log})
	engine.Handle(method, path, handler)

	rec := httptest.NewRecorder()
	engine.ServeHTTP(rec, httptest.NewRequest(method, path, nil))
	return rec
}

func decodeError(t *testing.T, rec *httptest.ResponseRecorder) httpx.ErrorDetail {
	t.Helper()

	var body httpx.ErrorBody
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("decode error body: %v", err)
	}
	return body.Error
}

func TestFailMapsKindToStatus(t *testing.T) {
	cases := []struct {
		err  error
		want int
	}{
		{errs.InvalidInput("bad"), http.StatusBadRequest},
		{errs.Unauthorized("no token"), http.StatusUnauthorized},
		{errs.Forbidden("not yours"), http.StatusForbidden},
		{errs.NotFound("gone"), http.StatusNotFound},
		{errs.Conflict("taken"), http.StatusConflict},
		{errs.Unavailable("down"), http.StatusServiceUnavailable},
		{errs.Internal("oops"), http.StatusInternalServerError},
		{errors.New("untyped"), http.StatusInternalServerError},
	}

	for _, tc := range cases {
		rec := serve(t, http.MethodGet, "/x", func(c *gin.Context) { httpx.Fail(c, tc.err) })
		if rec.Code != tc.want {
			t.Errorf("Fail(%v) status = %d, want %d", tc.err, rec.Code, tc.want)
		}
	}
}

func TestFailHidesInternalDetail(t *testing.T) {
	cause := errors.New(`pq: relation "orders" does not exist`)
	rec := serve(t, http.MethodGet, "/x", func(c *gin.Context) {
		httpx.Fail(c, errs.Internal("query failed").Wrap(cause).WithField("sql", "secret"))
	})

	detail := decodeError(t, rec)
	if detail.Message != "internal server error" {
		t.Fatalf("message = %q, want the generic message", detail.Message)
	}
	if detail.Fields != nil {
		t.Fatalf("fields = %v, want nil for an internal error", detail.Fields)
	}
	if strings.Contains(rec.Body.String(), "relation") {
		t.Fatalf("response leaked the cause: %s", rec.Body.String())
	}
}

func TestFailPassesThroughClientSafeDetail(t *testing.T) {
	rec := serve(t, http.MethodPost, "/x", func(c *gin.Context) {
		httpx.Fail(c, errs.InvalidInput("invalid reservation").
			WithCode("reservation_invalid").
			WithField("party_size", "must be at least 1"))
	})

	detail := decodeError(t, rec)
	if detail.Kind != string(errs.KindInvalidInput) {
		t.Errorf("kind = %q", detail.Kind)
	}
	if detail.Code != "reservation_invalid" {
		t.Errorf("code = %q", detail.Code)
	}
	if detail.Message != "invalid reservation" {
		t.Errorf("message = %q", detail.Message)
	}
	if detail.Fields["party_size"] != "must be at least 1" {
		t.Errorf("fields = %v", detail.Fields)
	}
}

func TestFailIncludesRequestID(t *testing.T) {
	rec := serve(t, http.MethodGet, "/x", func(c *gin.Context) {
		httpx.Fail(c, errs.NotFound("gone"))
	})

	header := rec.Header().Get(reqid.Header)
	if header == "" {
		t.Fatal("response is missing the request ID header")
	}
	if got := decodeError(t, rec).RequestID; got != header {
		t.Fatalf("body request_id = %q, header = %q; they must match", got, header)
	}
}

func TestRecoveryTurnsPanicIntoInternalError(t *testing.T) {
	rec := serve(t, http.MethodGet, "/x", func(c *gin.Context) {
		panic("handler bug")
	})

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want 500", rec.Code)
	}
	if detail := decodeError(t, rec); detail.Message != "internal server error" {
		t.Fatalf("message = %q; a panic must not leak its value", detail.Message)
	}
	if strings.Contains(rec.Body.String(), "handler bug") {
		t.Fatalf("response leaked the panic value: %s", rec.Body.String())
	}
}

func TestRequestIDAdoptsInboundHeader(t *testing.T) {
	log := logger.New(logger.Options{ServiceName: "test", Level: "error"})
	engine := httpx.NewEngine(httpx.ServerOptions{Log: log})

	var seen string
	engine.GET("/x", func(c *gin.Context) {
		seen = reqid.From(c.Request.Context())
		httpx.NoContent(c)
	})

	req := httptest.NewRequest(http.MethodGet, "/x", nil)
	req.Header.Set(reqid.Header, "caller-supplied-id")

	rec := httptest.NewRecorder()
	engine.ServeHTTP(rec, req)

	if seen != "caller-supplied-id" {
		t.Fatalf("context request id = %q, want the inbound one", seen)
	}
	if got := rec.Header().Get(reqid.Header); got != "caller-supplied-id" {
		t.Fatalf("echoed request id = %q", got)
	}
}
