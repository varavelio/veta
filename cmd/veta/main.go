package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/varavelio/veta/internal/cli"
)

// main owns process-level concerns: signal handling, final error logging, and
// the exit code.
//
// The application work is delegated to run so it can return errors instead of
// exiting directly. This keeps startup logic easier to test and lets main be the
// only place that decides how the process ends.
func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	if err := run(ctx); err != nil {
		slog.ErrorContext(ctx, "error running veta", slog.String("error", err.Error()))
		os.Exit(1)
	}
}

// run starts the application using the provided context.
//
// It contains the application workflow while main stays focused on process
// concerns. Keeping this logic outside main makes fatal failures explicit through
// returned errors and keeps process shutdown behavior separate from application
// startup behavior.
func run(ctx context.Context) (err error) {
	return cli.Run(ctx, os.Args[1:], os.Stdout, os.Stderr)
}
