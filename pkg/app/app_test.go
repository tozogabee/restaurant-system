package app_test

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"sync"
	"testing"
	"time"

	"github.com/tozogabee/restaurant-system/pkg/app"
)

func discardLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

func TestRunStopsEveryComponentOnCancel(t *testing.T) {
	runner := app.New(time.Second, discardLogger())

	var (
		mu      sync.Mutex
		stopped []string
	)
	record := func(name string) app.StopFunc {
		return func(context.Context) error {
			mu.Lock()
			defer mu.Unlock()
			stopped = append(stopped, name)
			return nil
		}
	}

	block := make(chan struct{})
	serve := func() error { <-block; return nil }

	runner.Add("http", serve, record("http"))
	runner.Add("grpc", serve, record("grpc"))
	runner.AddCloser("postgres", record("postgres"))

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	if err := runner.Run(ctx); err != nil {
		t.Fatalf("Run() = %v, want nil for a clean shutdown", err)
	}
	close(block)

	// Reverse registration order: dependents stop before their dependencies.
	want := []string{"postgres", "grpc", "http"}
	mu.Lock()
	defer mu.Unlock()
	if len(stopped) != len(want) {
		t.Fatalf("stopped = %v, want %v", stopped, want)
	}
	for i := range want {
		if stopped[i] != want[i] {
			t.Fatalf("stopped = %v, want %v", stopped, want)
		}
	}
}

func TestRunReturnsComponentFailureAndStillShutsDown(t *testing.T) {
	runner := app.New(time.Second, discardLogger())

	failure := errors.New("port already in use")
	stopped := make(chan struct{})

	runner.Add("http", func() error { return failure }, nil)
	runner.AddCloser("postgres", func(context.Context) error {
		close(stopped)
		return nil
	})

	err := runner.Run(context.Background())
	if !errors.Is(err, failure) {
		t.Fatalf("Run() = %v, want it to wrap %v", err, failure)
	}

	select {
	case <-stopped:
	case <-time.After(time.Second):
		t.Fatal("a failing component must still trigger shutdown of the rest")
	}
}

func TestRunReportsShutdownFailure(t *testing.T) {
	runner := app.New(time.Second, discardLogger())

	stopErr := errors.New("drain timed out")
	runner.AddCloser("postgres", func(context.Context) error { return stopErr })

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	if err := runner.Run(ctx); !errors.Is(err, stopErr) {
		t.Fatalf("Run() = %v, want it to wrap %v", err, stopErr)
	}
}

func TestShutdownTimeoutBoundsStop(t *testing.T) {
	runner := app.New(50*time.Millisecond, discardLogger())

	deadlineSeen := make(chan bool, 1)
	runner.AddCloser("slow", func(ctx context.Context) error {
		_, ok := ctx.Deadline()
		deadlineSeen <- ok
		return nil
	})

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_ = runner.Run(ctx)

	if !<-deadlineSeen {
		t.Fatal("stop context has no deadline; a slow component could hang shutdown")
	}
}
