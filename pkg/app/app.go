// Package app holds the process lifecycle helpers used by every main package.
package app

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"golang.org/x/sync/errgroup"
)

// SignalContext returns a context cancelled on SIGINT or SIGTERM.
func SignalContext() (context.Context, context.CancelFunc) {
	return signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
}

// HandleHealthcheck exits with probe's result when the binary is invoked as
// `<binary> healthcheck`. It is used by the Docker HEALTHCHECK.
func HandleHealthcheck(probe func() int) {
	if len(os.Args) > 1 && os.Args[1] == "healthcheck" {
		os.Exit(probe())
	}
}

// Run starts every component and blocks until ctx is cancelled or one of
// them fails, in which case the others are cancelled too.
func Run(ctx context.Context, components ...func(context.Context) error) error {
	g, ctx := errgroup.WithContext(ctx)
	for _, run := range components {
		g.Go(func() error { return run(ctx) })
	}
	return g.Wait()
}
