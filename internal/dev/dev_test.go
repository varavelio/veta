package dev

import (
	"context"
	"errors"
	"net"
	"net/http"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/varavelio/veta/internal/build"
)

// TestHTTPServerShutdownCompletesPromptly verifies the development HTTP server
// serves the live-reload endpoint and shuts down without waiting on idle
// connections.
func TestHTTPServerShutdownCompletesPromptly(t *testing.T) {
	revision := newRevision()
	server, cancelRequests := newHTTPServer(t.Context(), revision)
	listenConfig := net.ListenConfig{}
	listener, err := listenConfig.Listen(t.Context(), "tcp", "127.0.0.1:0")
	require.NoError(t, err)
	serveErrors := make(chan error, 1)
	go func() {
		serveErrors <- server.Serve(listener)
	}()

	request, err := http.NewRequestWithContext(
		t.Context(),
		http.MethodGet,
		"http://"+listener.Addr().String(),
		nil,
	)
	require.NoError(t, err)
	response, err := http.DefaultClient.Do(request)
	require.NoError(t, err)
	defer func() {
		_ = response.Body.Close()
	}()
	require.Equal(t, http.StatusOK, response.StatusCode)
	require.Equal(t, "application/json", response.Header.Get("Content-Type"))

	started := time.Now()
	cancelRequests()
	require.NoError(t, shutdownHTTPServer(server))
	require.Less(t, time.Since(started), time.Second)

	select {
	case err := <-serveErrors:
		require.True(t, err == nil || errors.Is(err, http.ErrServerClosed))
	case <-time.After(time.Second):
		t.Fatal("HTTP server did not stop")
	}
}

// TestRunRebuildsBuildsOnChange verifies a change triggers one build whose
// success is delivered through onBuilt.
func TestRunRebuildsBuildsOnChange(t *testing.T) {
	runner := newFakeBuilder()
	changes := make(chan struct{}, 4)
	runRebuildsAsync(t.Context(), changes, runner)

	changes <- struct{}{}
	requireStartedBuilds(t, runner, 1)
	runner.release <- struct{}{}
	requireBuiltCount(t, runner, 1)
	require.Equal(t, 1, runner.startedCount(), "unexpected extra builds")
	require.Zero(t, runner.errorCount(), "unexpected build error")
}

// TestRunRebuildsCancelsInFlightBuild verifies a change during a build cancels
// it and immediately starts a fresh build without reporting the canceled error.
func TestRunRebuildsCancelsInFlightBuild(t *testing.T) {
	runner := newFakeBuilder()
	changes := make(chan struct{}, 4)
	runRebuildsAsync(t.Context(), changes, runner)

	changes <- struct{}{}
	requireStartedBuilds(t, runner, 1)

	changes <- struct{}{}
	require.Eventually(
		t,
		func() bool { return runner.canceledCount() == 1 },
		time.Second,
		5*time.Millisecond,
	)
	requireStartedBuilds(t, runner, 2)

	runner.release <- struct{}{}
	requireBuiltCount(t, runner, 1)
	require.Zero(t, runner.errorCount(), "canceled build must not report an error")
}

// TestRunRebuildsReportsBuildErrors verifies a failed build is surfaced through
// onError.
func TestRunRebuildsReportsBuildErrors(t *testing.T) {
	runner := newFakeBuilder()
	runner.fail = true
	changes := make(chan struct{}, 4)
	runRebuildsAsync(t.Context(), changes, runner)

	changes <- struct{}{}
	requireStartedBuilds(t, runner, 1)
	runner.release <- struct{}{}
	require.Eventually(
		t,
		func() bool { return runner.errorCount() == 1 },
		time.Second,
		5*time.Millisecond,
	)
}

// TestRunRebuildsReturnsServerError verifies a fatal server error ends the loop.
func TestRunRebuildsReturnsServerError(t *testing.T) {
	runner := newFakeBuilder()
	serverErrors := make(chan error, 1)
	serverErrors <- errors.New("boom")

	done := make(chan error, 1)
	go func() {
		done <- runRebuilds(
			t.Context(),
			make(chan struct{}),
			serverErrors,
			make(chan error),
			runner.build,
			func(result build.Result) {},
			func(error) {},
		)
	}()

	select {
	case err := <-done:
		require.EqualError(t, err, "boom")
	case <-time.After(time.Second):
		t.Fatal("runRebuilds did not return on server error")
	}
	require.Zero(t, runner.startedCount(), "no builds should run before the fatal error")
}

// TestRunRebuildsStopsOnContextCancel verifies canceling the context stops the
// loop and cancels any in-flight build.
func TestRunRebuildsStopsOnContextCancel(t *testing.T) {
	runner := newFakeBuilder()
	changes := make(chan struct{}, 4)
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()

	done := make(chan error, 1)
	go func() {
		done <- runRebuilds(
			ctx,
			changes,
			make(chan error),
			make(chan error),
			runner.build,
			func(result build.Result) {},
			func(error) {},
		)
	}()

	changes <- struct{}{}
	requireStartedBuilds(t, runner, 1)

	cancel()
	select {
	case err := <-done:
		require.NoError(t, err)
	case <-time.After(time.Second):
		t.Fatal("runRebuilds did not stop on context cancel")
	}
	require.Eventually(
		t,
		func() bool { return runner.canceledCount() == 1 },
		time.Second,
		5*time.Millisecond,
	)
}

// fakeBuilder simulates cancelable builds controlled by the test.
type fakeBuilder struct {
	started chan struct{}
	release chan struct{}
	fail    bool

	mu         sync.Mutex
	startCount int
	canceled   int
	successes  int
	failures   int
}

// newFakeBuilder creates a fake builder whose builds wait until released.
func newFakeBuilder() *fakeBuilder {
	return &fakeBuilder{
		started: make(chan struct{}, 16),
		release: make(chan struct{}, 16),
	}
}

// build waits for release or cancellation, then reports a configurable result.
func (builder *fakeBuilder) build(ctx context.Context) (build.Result, error) {
	buildCtx, cancel := context.WithCancel(ctx)
	defer cancel()
	builder.mu.Lock()
	builder.startCount++
	builder.mu.Unlock()

	builder.started <- struct{}{}

	select {
	case <-buildCtx.Done():
		builder.recordCanceled()
		return build.Result{}, buildCtx.Err()
	case <-builder.release:
	}
	if err := buildCtx.Err(); err != nil {
		builder.recordCanceled()
		return build.Result{}, err
	}

	builder.mu.Lock()
	if builder.fail {
		builder.failures++
		builder.mu.Unlock()
		return build.Result{}, errors.New("build failed")
	}
	builder.successes++
	builder.mu.Unlock()

	return build.Result{OutputDir: "out"}, nil
}

// recordCanceled marks one build context cancellation.
func (builder *fakeBuilder) recordCanceled() {
	builder.mu.Lock()
	builder.canceled++
	builder.mu.Unlock()
}

// canceledCount returns the number of builds canceled while running.
func (builder *fakeBuilder) canceledCount() int {
	builder.mu.Lock()
	defer builder.mu.Unlock()
	return builder.canceled
}

// startedCount returns the number of started builds.
func (builder *fakeBuilder) startedCount() int {
	builder.mu.Lock()
	defer builder.mu.Unlock()
	return builder.startCount
}

// builtCount returns the number of successful builds.
func (builder *fakeBuilder) builtCount() int {
	builder.mu.Lock()
	defer builder.mu.Unlock()
	return builder.successes
}

// errorCount returns the number of reported build errors.
func (builder *fakeBuilder) errorCount() int {
	builder.mu.Lock()
	defer builder.mu.Unlock()
	return builder.failures
}

// runRebuildsAsync starts runRebuilds with a recording builder and empty event
// channels.
func runRebuildsAsync(
	ctx context.Context,
	changes chan struct{},
	runner *fakeBuilder,
) <-chan error {
	done := make(chan error, 1)
	go func() {
		done <- runRebuilds(
			ctx,
			changes,
			make(chan error),
			make(chan error),
			runner.build,
			func(result build.Result) {},
			func(error) {},
		)
	}()
	return done
}

// requireStartedBuilds waits until the fake builder started want builds.
func requireStartedBuilds(t *testing.T, builder *fakeBuilder, want int) {
	t.Helper()
	require.Eventually(t, func() bool {
		return builder.startedCount() == want
	}, time.Second, 5*time.Millisecond)
}

// requireBuiltCount waits until the fake builder completed want builds.
func requireBuiltCount(t *testing.T, builder *fakeBuilder, want int) {
	t.Helper()
	require.Eventually(t, func() bool {
		return builder.builtCount() == want
	}, time.Second, 5*time.Millisecond)
}
