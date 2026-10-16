package health_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/tozogabee/restaurant-system/pkg/health"
)

func TestMain(m *testing.M) {
	gin.SetMode(gin.TestMode)
	m.Run()
}

func mounted(c *health.Checker) *gin.Engine {
	engine := gin.New()
	c.Mount(engine)
	return engine
}

func get(engine *gin.Engine, path string) *httptest.ResponseRecorder {
	rec := httptest.NewRecorder()
	engine.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, path, nil))
	return rec
}

func TestLivenessIgnoresFailingDependencies(t *testing.T) {
	c := health.New(time.Second)
	c.Register("postgres", func(context.Context) error { return errors.New("down") })

	// A failing dependency should take the service out of the load balancer,
	// not get the container restarted.
	if rec := get(mounted(c), "/healthz"); rec.Code != http.StatusOK {
		t.Fatalf("/healthz status = %d, want 200 even with a failing check", rec.Code)
	}
}

func TestReadinessReportsEachCheck(t *testing.T) {
	c := health.New(time.Second)
	c.Register("postgres", func(context.Context) error { return nil })
	c.Register("redis", func(context.Context) error { return errors.New("connection refused") })

	rec := get(mounted(c), "/readyz")
	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want 503 while a check fails", rec.Code)
	}

	var status health.Status
	if err := json.NewDecoder(rec.Body).Decode(&status); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if status.Status != "degraded" {
		t.Errorf("status = %q, want %q", status.Status, "degraded")
	}
	if status.Checks["postgres"] != "ok" {
		t.Errorf("checks[postgres] = %q", status.Checks["postgres"])
	}
	if status.Checks["redis"] != "connection refused" {
		t.Errorf("checks[redis] = %q, want the error message", status.Checks["redis"])
	}
}

func TestReadinessOKWhenAllChecksPass(t *testing.T) {
	c := health.New(time.Second)
	c.Register("postgres", func(context.Context) error { return nil })
	c.Register("redis", func(context.Context) error { return nil })

	rec := get(mounted(c), "/readyz")
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
}

func TestChecksRunUnderTheCheckerTimeout(t *testing.T) {
	c := health.New(30 * time.Millisecond)
	c.Register("hang", func(ctx context.Context) error {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(2 * time.Second):
			return nil
		}
	})

	start := time.Now()
	rec := get(mounted(c), "/readyz")

	if elapsed := time.Since(start); elapsed > time.Second {
		t.Fatalf("probe took %s; a hanging check must not hang the probe", elapsed)
	}
	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want 503", rec.Code)
	}
}
