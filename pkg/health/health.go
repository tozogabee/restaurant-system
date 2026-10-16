// Package health exposes the liveness and readiness endpoints that the local
// compose stack and, later, Kubernetes use to decide whether a service should
// receive traffic.
package health

import (
	"context"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

// Check reports whether one dependency is usable. It must be cheap: probes run
// often, and a check that queries real data turns a probe into load.
type Check func(ctx context.Context) error

// Checker collects the named checks a service depends on.
type Checker struct {
	mu     sync.RWMutex
	checks map[string]Check
	// timeout bounds the whole readiness probe, not each check.
	timeout time.Duration
}

// New returns an empty Checker. A zero timeout defaults to 2s.
func New(timeout time.Duration) *Checker {
	if timeout == 0 {
		timeout = 2 * time.Second
	}
	return &Checker{checks: make(map[string]Check), timeout: timeout}
}

// Register adds a named check. Registering the same name twice replaces it.
func (c *Checker) Register(name string, check Check) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.checks[name] = check
}

// Status is the readiness response body.
type Status struct {
	// Status is "ok" when every check passed, "degraded" otherwise.
	Status string `json:"status"`
	// Checks maps each dependency name to "ok" or its error message.
	Checks map[string]string `json:"checks"`
}

// Run executes every check in parallel and reports the combined result.
func (c *Checker) Run(ctx context.Context) Status {
	c.mu.RLock()
	checks := make(map[string]Check, len(c.checks))
	for name, check := range c.checks {
		checks[name] = check
	}
	c.mu.RUnlock()

	ctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	results := make(map[string]string, len(checks))
	var (
		mu sync.Mutex
		wg sync.WaitGroup
	)

	for name, check := range checks {
		wg.Add(1)
		go func(name string, check Check) {
			defer wg.Done()

			result := "ok"
			if err := check(ctx); err != nil {
				result = err.Error()
			}

			mu.Lock()
			results[name] = result
			mu.Unlock()
		}(name, check)
	}
	wg.Wait()

	status := Status{Status: "ok", Checks: results}
	for _, result := range results {
		if result != "ok" {
			status.Status = "degraded"
			break
		}
	}
	return status
}

// Mount registers the probe endpoints on the given router.
//
// /healthz answers as long as the process is running: a failing dependency
// should take the service out of the load balancer, not get it restarted.
// /readyz runs the checks and answers 503 while any of them fails.
func (c *Checker) Mount(router gin.IRouter) {
	router.GET("/healthz", func(ctx *gin.Context) {
		ctx.JSON(http.StatusOK, Status{Status: "ok"})
	})

	router.GET("/readyz", func(ctx *gin.Context) {
		status := c.Run(ctx.Request.Context())

		code := http.StatusOK
		if status.Status != "ok" {
			code = http.StatusServiceUnavailable
		}
		ctx.JSON(code, status)
	})
}
