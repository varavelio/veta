package dev

import (
	"errors"
	"io"
	"net"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// TestHTTPServerShutdownClosesActiveSSE verifies server cancellation releases
// long-lived live-reload requests before graceful shutdown waits for them.
func TestHTTPServerShutdownClosesActiveSSE(t *testing.T) {
	broadcaster := newBroadcaster()
	server, cancelRequests := newHTTPServer(t.Context(), broadcaster)
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
	require.Eventually(t, func() bool {
		return broadcaster.clientCount() == 1
	}, time.Second, 10*time.Millisecond)

	started := time.Now()
	cancelRequests()
	require.NoError(t, shutdownHTTPServer(server))
	require.Less(t, time.Since(started), time.Second)

	streamClosed := make(chan struct{})
	go func() {
		_, _ = io.Copy(io.Discard, response.Body)
		close(streamClosed)
	}()
	select {
	case <-streamClosed:
	case <-time.After(time.Second):
		t.Fatal("SSE stream did not close")
	}
	require.Eventually(t, func() bool {
		return broadcaster.clientCount() == 0
	}, time.Second, 10*time.Millisecond)
	select {
	case err := <-serveErrors:
		require.True(t, err == nil || errors.Is(err, http.ErrServerClosed))
	case <-time.After(time.Second):
		t.Fatal("HTTP server did not stop")
	}
}
